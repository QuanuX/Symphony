#include "connector.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/operation.hpp"
#include "symphony/knowledge/engine/temporal.hpp"
#include <duckdb.h>
#include <algorithm>
#include <atomic>
#include <cerrno>
#include <chrono>
#include <cstring>
#include <fcntl.h>
#include <filesystem>
#include <map>
#include <memory>
#include <regex>
#include <set>
#include <string>
#include <sys/file.h>
#include <sys/stat.h>
#include <thread>
#include <unistd.h>
#include <vector>

namespace symphony::scv::duckdb_connector {
namespace {
[[noreturn]] void fail(const std::string& code, const std::string& message) { throw engine::Error("connector." + code, message, 4); }
void require(bool okay, const std::string& message) { if (!okay) fail("invalid", message); }
void fields(const Json& value, const std::set<std::string>& required, const std::set<std::string>& optional = {}) {
    require(value.is_object(), "object required");
    for (const auto& key : required) require(value.contains(key), "required field absent: " + key);
    for (const auto& [key, unused] : value.items()) { static_cast<void>(unused); require(required.contains(key) || optional.contains(key), "unexpected field: " + key); }
}
std::string text(const Json& value, std::size_t maximum = 512) {
    require(value.is_string(), "string required");
    auto result = value.get<std::string>();
    require(!result.empty() && result.size() <= maximum && result.find('\0') == std::string::npos, "string outside bound");
    return result;
}
bool is_digest(const std::string& value) { return value.size() == 71 && value.starts_with("sha256:") && std::all_of(value.begin() + 7, value.end(), [](char ch) { return (ch >= 'a' && ch <= 'f') || (ch >= '0' && ch <= '9'); }); }
std::string digest(const Json& value) { auto result = text(value, 71); require(is_digest(result), "tagged SHA256 required"); return result; }
Json seal(Json value, const char* field = "digest") { value[field] = engine::tagged_sha256(value.dump()); return value; }
void verify_seal(Json value, const char* field = "digest") { require(value.is_object() && value.contains(field), "sealed object required"); auto expected = digest(value[field]); value.erase(field); require(engine::tagged_sha256(value.dump()) == expected, "sealed object digest mismatch"); }
void deadline(const engine::Request& request) { if (engine::unix_time_ms() >= request.deadline_unix_ms) fail("deadline", "request deadline exceeded"); }
std::string token(const Json& value) { auto s = text(value, 128); require(std::regex_match(s, std::regex("[A-Za-z0-9][A-Za-z0-9._:-]{0,127}")), "namespace syntax differs"); return s; }
std::string operation_id(const Json& value) { auto s = text(value, 128); require(std::regex_match(s,std::regex("[A-Za-z0-9._:-]{1,128}")), "operation ID syntax differs"); return s; }
std::string tops_id(const Json& value) { auto s = text(value, 36); require(std::regex_match(s, std::regex("[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}")), "canonical TOPS UUID required"); return s; }
void scope(const Json& value) { static_cast<void>(tops_id(value.at("tops_id"))); static_cast<void>(token(value.at("namespace"))); }
void installation(const Json& value, bool connector) {
    fields(value, {"Role","ModuleID","EngineID","Version","Prefix","ReceiptPath","ReceiptDigest","ReceiptProtocol","ExecutablePath","ExecutableDigest"});
    for (const auto* key : {"Role","ModuleID","EngineID","Version"}) static_cast<void>(text(value.at(key),128));
    for (const auto* key : {"Prefix","ReceiptPath","ExecutablePath"}) {
        auto path = text(value.at(key),4096); require(path.starts_with('/') && path.find("//") == std::string::npos && std::filesystem::path(path).lexically_normal().string() == path && (path == "/" || !path.ends_with('/')), "installation path is not clean absolute");
    }
    static_cast<void>(digest(value.at("ReceiptDigest"))); static_cast<void>(digest(value.at("ExecutableDigest")));
    require(value.at("ReceiptProtocol") == "symphony.knowledge.install-receipt.v2", "receipt protocol differs");
    if (connector) require(value.at("Role") == "scv-graph-duckdb-connector" && value.at("ModuleID") == "scv-graph-duckdb-connector" && value.at("EngineID") == engine_id && value.at("Version") == version, "connector identity differs");
    else {
        const std::set<std::string> domains={"scv","schv","scev","schv-aws","schv-azure","schv-do","schv-gcp","scev-cf"};
        const std::set<std::string> versions={"0.1.0-dev","0.2.0-dev","0.3.0-dev","0.4.0-dev","0.5.0-dev","0.6.0-dev","0.7.0-dev","0.8.0-dev","0.9.0-dev","0.10.0-dev"};
        require(domains.contains(text(value.at("Role"))) && versions.contains(text(value.at("Version"))) && value.at("ModuleID") == text(value.at("Role")) + "-engine" && value.at("EngineID") == "symphony-" + text(value.at("Role")), "owner identity differs");
    }
    const auto prefix = text(value.at("Prefix"),4096), module = text(value.at("ModuleID")), release = text(value.at("Version"));
    require(value.at("ReceiptPath") == (std::filesystem::path(prefix) / "share/symphony/receipts" / module / release / "install-receipt.json").string(), "receipt path identity differs");
    require(value.at("ExecutablePath") == (std::filesystem::path(prefix) / "libexec/symphony" / module / release / text(value.at("EngineID"))).string(), "executable path identity differs");
}
void scope_filter(const Json& value) {
    require(value.is_object() && value.size() <= 16,"scope shape differs");
    for(const auto& [key,item]:value.items()) require(!key.empty() && key.size() <= 128 && item.is_string() && item.get_ref<const std::string&>().size() <= 512,"scope qualifier bound differs");
}
Json projection(const Json& graph) {
    fields(graph,{"protocol","domain","selection_policy","knowledge_digests","interpretations","captures","claims","native_nodes","native_edges","limitations","digest"});
    require(graph.at("protocol") == "symphony.scv.graph.v1", "graph protocol differs"); verify_seal(graph);
    for(const auto* key:{"captures","knowledge_digests","interpretations"}) require(graph.at(key).is_array() && graph.at(key).size()<=16,"graph collection bound exceeded");
    Json result = {{"claims",Json::array()},{"nodes",Json::array()},{"edges",Json::array()}};
    for (const auto& [kind, field, key, maximum] : std::vector<std::tuple<std::string,std::string,std::string,std::size_t>>{{"claims","claims","claim_id",128},{"nodes","native_nodes","node_id",512},{"edges","native_edges","",1024}}) {
        const auto& values = graph.at(field); require(values.is_array() && values.size() <= maximum,"graph collection bound exceeded");
        std::map<std::string,Json> rows;
        for (const auto& value : values) {
            require(value.is_object(),"graph row object required");
            auto id = key.empty() ? engine::tagged_sha256(value.dump()) : text(value.at(key));
            if (kind == "claims") { static_cast<void>(text(value.at("subject"))); static_cast<void>(text(value.at("predicate"))); require(value.contains("scope"),"claim scope absent"); scope_filter(value.at("scope")); }
            if (kind == "nodes") { static_cast<void>(text(value.at("kind"))); static_cast<void>(digest(value.at("capture_digest"))); }
            if (kind == "edges") { fields(value,{"from","relation","to"}); for (const auto* name : {"from","relation","to"}) static_cast<void>(text(value.at(name))); }
            require(rows.emplace(id,value).second,"duplicate graph row key");
        }
        for (const auto& [id,value] : rows) result[kind].push_back({{"key",id},{"value",value}});
    }
    return result;
}
Json counts(const Json& projected) { return {{"claims",projected.at("claims").size()},{"nodes",projected.at("nodes").size()},{"edges",projected.at("edges").size()}}; }
void validate_snapshot(const Json& value) {
    fields(value,{"protocol","backend","mapping_version","tops_id","namespace","graph","owner","connector","digest"});
    require(value.at("protocol") == "symphony.scv.graph-index-snapshot.v1" && value.at("backend") == "duckdb" && value.at("mapping_version") == "1","snapshot protocol/backend differs");
    scope(value); installation(value.at("owner"),false); installation(value.at("connector"),true); verify_seal(value);
    require(value.at("graph").at("domain") == value.at("owner").at("Role"),"graph owner domain differs"); static_cast<void>(projection(value.at("graph")));
}
void validate_intent(const Json& value) {
    fields(value,{"protocol","operation_id","snapshot","validation_query_time","digest"});
    require(value.at("protocol") == "symphony.scv.graph-index-intent.v1","intent protocol differs");
    static_cast<void>(operation_id(value.at("operation_id"))); auto time = text(value.at("validation_query_time"),20);
    require(engine::is_utc_seconds(time),"query time shape differs");
    validate_snapshot(value.at("snapshot")); verify_seal(value);
}
struct File final { int fd = -1; explicit File(int handle = -1) : fd(handle) {} ~File() { if (fd >= 0) ::close(fd); } File(const File&) = delete; File& operator=(const File&) = delete; };
void private_file(int fd, bool directory) {
    struct stat st{}; require(fd >= 0 && ::fstat(fd,&st) == 0,"private file unavailable");
    require(st.st_uid == ::geteuid() && (st.st_mode & 07777) == (directory ? 0700 : 0600),"private file ownership/permissions differ");
    require(directory ? S_ISDIR(st.st_mode) : (S_ISREG(st.st_mode) && st.st_nlink == 1),"private regular file required");
    if (!directory) require(st.st_size >= 0 && st.st_size <= (512LL << 20),"database file bound exceeded");
}
bool child(int root,const char* name) {
    struct stat st{}; if (::fstatat(root,name,&st,AT_SYMLINK_NOFOLLOW) != 0) { if (errno == ENOENT) return false; fail("storage","database child cannot be inspected"); }
    require(S_ISREG(st.st_mode) && st.st_nlink == 1 && st.st_uid == ::geteuid() && (st.st_mode & 07777) == 0600 && st.st_size <= (512LL << 20),"database child is not an owned private regular file"); return true;
}
struct Result final {
    duckdb_result value{};
    std::vector<duckdb_data_chunk> chunks;
    std::size_t rows = 0; bool loaded = false;
    ~Result() { for(auto& chunk:chunks) duckdb_destroy_data_chunk(&chunk); duckdb_destroy_result(&value); }
    Result() = default; Result(const Result&) = delete; Result& operator=(const Result&) = delete;
    void load() {
        if(loaded) return; loaded=true;
        const auto count=duckdb_result_chunk_count(value); require(count<=2048,"stored result chunk bound exceeded");
        for(idx_t i=0;i<count;++i) { auto chunk=duckdb_result_get_chunk(value,i); require(chunk!=nullptr,"stored result chunk unavailable"); chunks.push_back(chunk); rows+=duckdb_data_chunk_get_size(chunk); require(rows<=2048,"stored result row bound exceeded"); }
    }
    std::size_t size() { load(); return rows; }
    std::string string(std::size_t row,std::size_t column) {
        load(); require(row<rows && duckdb_column_type(&value,column)==DUCKDB_TYPE_VARCHAR,"stored text type differs");
        for(auto chunk:chunks) {
            const auto count=duckdb_data_chunk_get_size(chunk); if(row>=count) { row-=count; continue; }
            const auto vector=duckdb_data_chunk_get_vector(chunk,column); require(vector!=nullptr,"stored vector unavailable");
            const auto validity=duckdb_vector_get_validity(vector); require(validity==nullptr || duckdb_validity_row_is_valid(validity,row),"stored text is null");
            auto data=static_cast<duckdb_string_t*>(duckdb_vector_get_data(vector)); require(data!=nullptr,"stored text unavailable");
            const auto size=duckdb_string_t_length(data[row]); require(size<=(4U<<20),"stored text bound exceeded");
            return std::string(duckdb_string_t_data(&data[row]),size);
        }
        fail("storage","stored text row unavailable");
    }
};
struct Statement final {
    duckdb_prepared_statement value = nullptr; const engine::Request& request;
    Statement(duckdb_connection con,const engine::Request& req,const std::string& sql) : request(req) {
        deadline(request); if (duckdb_prepare(con,sql.c_str(),&value) != DuckDBSuccess) { duckdb_destroy_prepare(&value); fail("storage","statement preparation failed"); }
    }
    ~Statement() { duckdb_destroy_prepare(&value); }
    void bind(std::size_t index,const std::string& value_) { require(duckdb_bind_varchar_length(value,index,value_.data(),value_.size()) == DuckDBSuccess,"parameter binding failed"); }
    std::unique_ptr<Result> run() {
        deadline(request); auto out = std::make_unique<Result>();
        if (duckdb_execute_prepared(value,&out->value) != DuckDBSuccess) { deadline(request); fail("storage","database statement failed"); }
        deadline(request); return out;
    }
};
const std::vector<std::string> schema_sql = {
 "CREATE TABLE metadata (key VARCHAR PRIMARY KEY, value VARCHAR NOT NULL)",
 "CREATE TABLE intents (tops_id VARCHAR, namespace VARCHAR, operation_id VARCHAR, intent_digest VARCHAR NOT NULL, snapshot_digest VARCHAR NOT NULL, state VARCHAR NOT NULL, document VARCHAR NOT NULL, PRIMARY KEY(tops_id,namespace,operation_id))",
 "CREATE TABLE snapshots (tops_id VARCHAR, namespace VARCHAR, snapshot_digest VARCHAR, document VARCHAR NOT NULL, PRIMARY KEY(tops_id,namespace,snapshot_digest))",
 "CREATE TABLE claims (tops_id VARCHAR, namespace VARCHAR, snapshot_digest VARCHAR, row_key VARCHAR, value VARCHAR NOT NULL, claim_id VARCHAR NOT NULL, subject VARCHAR NOT NULL, predicate VARCHAR NOT NULL, scope VARCHAR NOT NULL, PRIMARY KEY(tops_id,namespace,snapshot_digest,row_key))",
 "CREATE TABLE nodes (tops_id VARCHAR, namespace VARCHAR, snapshot_digest VARCHAR, row_key VARCHAR, value VARCHAR NOT NULL, node_id VARCHAR NOT NULL, kind VARCHAR NOT NULL, capture_digest VARCHAR NOT NULL, PRIMARY KEY(tops_id,namespace,snapshot_digest,row_key))",
 "CREATE TABLE edges (tops_id VARCHAR, namespace VARCHAR, snapshot_digest VARCHAR, row_key VARCHAR, value VARCHAR NOT NULL, from_id VARCHAR NOT NULL, relation VARCHAR NOT NULL, to_id VARCHAR NOT NULL, PRIMARY KEY(tops_id,namespace,snapshot_digest,row_key))",
 "CREATE INDEX claims_subject ON claims(subject)","CREATE INDEX claims_predicate ON claims(predicate)","CREATE INDEX claims_scope ON claims(scope)",
 "CREATE INDEX nodes_kind ON nodes(kind)","CREATE INDEX nodes_capture ON nodes(capture_digest)",
 "CREATE INDEX edges_from ON edges(from_id)","CREATE INDEX edges_relation ON edges(relation)","CREATE INDEX edges_to ON edges(to_id)"
};
const std::map<std::string,std::vector<std::string>> indexed_fields = {{"claims",{"claim_id","subject","predicate","scope"}},{"nodes",{"node_id","kind","capture_digest"}},{"edges",{"from","relation","to"}}};
std::string sql_column(const std::string& field) { return field == "from" ? "from_id" : field == "to" ? "to_id" : field; }
struct Database final {
    const engine::Request& request; File root; File lock; duckdb_database database = nullptr; duckdb_connection connection = nullptr; std::jthread watcher; bool transaction = false;
    explicit Database(const engine::Request& req,bool create) : request(req),root(::open(".",O_RDONLY|O_DIRECTORY|O_NOFOLLOW|O_CLOEXEC)) {
        private_file(root.fd,true); child(root.fd,"connector.lock");
        lock.fd = ::openat(root.fd,"connector.lock",O_RDWR|O_NOFOLLOW|O_CLOEXEC|(create?O_CREAT:0),0600); private_file(lock.fd,false);
        while (::flock(lock.fd,LOCK_EX|LOCK_NB) != 0) { if (errno != EWOULDBLOCK && errno != EAGAIN) fail("storage","connector lock unavailable"); deadline(request); ::usleep(1000); }
        const bool exists = child(root.fd,"index.duckdb"); require(create || exists,"database is absent");
        child(root.fd,"index.duckdb.wal");
        struct stat temp{}; require(::fstatat(root.fd,"index.duckdb.tmp",&temp,AT_SYMLINK_NOFOLLOW) != 0 && errno == ENOENT,"temporary directory is unsupported");
        require(std::string(duckdb_library_version()) == "v1.5.5","exact linked DuckDB release differs");
        ::umask(0077);
        duckdb_config config = nullptr;
        require(duckdb_create_config(&config) == DuckDBSuccess,"database configuration unavailable");
        try {
            for (const auto& [key,value] : std::vector<std::pair<const char*,const char*>>{{"threads","1"},{"memory_limit","256MB"},{"temp_directory",""},{"max_temp_directory_size","0B"},{"enable_external_access","false"},{"autoinstall_known_extensions","false"},{"autoload_known_extensions","false"},{"allow_unsigned_extensions","false"},{"allow_community_extensions","false"},{"allow_persistent_secrets","false"},{"allocator_background_threads","false"}})
                require(duckdb_set_config(config,key,value) == DuckDBSuccess,std::string("database configuration rejected: ") + key);
            char* error = nullptr; auto opened = duckdb_open_ext("index.duckdb",&database,config,&error); duckdb_free(error); require(opened == DuckDBSuccess,"database open failed");
            duckdb_destroy_config(&config); require(duckdb_connect(database,&connection) == DuckDBSuccess,"database connection failed");
            watcher = std::jthread([this](std::stop_token stop){ while (!stop.stop_requested()) { if (engine::unix_time_ms() >= request.deadline_unix_ms) { duckdb_interrupt(connection); return; } std::this_thread::sleep_for(std::chrono::milliseconds(1)); } });
            require(child(root.fd,"index.duckdb"),"database identity unavailable");
            exec("SET lock_configuration=true");
            if (!exists) initialize();
            validate_schema(); deadline(request);
        } catch (...) { duckdb_destroy_config(&config); close(); throw; }
    }
    void close() { watcher.request_stop(); if (watcher.joinable()) watcher.join(); if (connection) { if (transaction) { duckdb_result result{}; duckdb_query(connection,"ROLLBACK",&result); duckdb_destroy_result(&result); } duckdb_disconnect(&connection); } if (database) duckdb_close(&database); }
    ~Database() { close(); }
    std::unique_ptr<Result> query(const std::string& sql,const std::vector<std::string>& parameters = {}) { Statement statement(connection,request,sql); for (std::size_t i=0;i<parameters.size();++i) statement.bind(i+1,parameters[i]); return statement.run(); }
    void exec(const std::string& sql,const std::vector<std::string>& parameters = {}) { static_cast<void>(query(sql,parameters)); }
    void begin() { exec("BEGIN TRANSACTION"); transaction = true; }
    void commit() { deadline(request); exec("COMMIT"); transaction = false; require(child(root.fd,"index.duckdb"),"database file disappeared"); child(root.fd,"index.duckdb.wal"); if (::fsync(root.fd) != 0) fail("storage","database directory synchronization failed"); }
    void initialize() {
        begin(); for (const auto& sql : schema_sql) exec(sql);
        exec("INSERT INTO metadata VALUES (?,?)",{"identity","symphony.scv.graph-index.duckdb.v1"});
        exec("INSERT INTO metadata VALUES (?,?)",{"schema_digest",engine::tagged_sha256(Json(schema_sql).dump())}); commit();
    }
    void validate_schema() {
        auto meta = query("SELECT key,value FROM metadata ORDER BY key LIMIT 3");
        require(meta->size() == 2 && meta->string(0,0) == "identity" && meta->string(0,1) == "symphony.scv.graph-index.duckdb.v1" && meta->string(1,0) == "schema_digest" && meta->string(1,1) == engine::tagged_sha256(Json(schema_sql).dump()),"database schema identity differs");
        // Validate the complete table/column inventory independently of the mutable identity row.
        const std::map<std::string,std::vector<std::string>> expected = {
            {"metadata",{"key","value"}}, {"intents",{"tops_id","namespace","operation_id","intent_digest","snapshot_digest","state","document"}},
            {"snapshots",{"tops_id","namespace","snapshot_digest","document"}},
            {"claims",{"tops_id","namespace","snapshot_digest","row_key","value","claim_id","subject","predicate","scope"}},
            {"nodes",{"tops_id","namespace","snapshot_digest","row_key","value","node_id","kind","capture_digest"}},
            {"edges",{"tops_id","namespace","snapshot_digest","row_key","value","from_id","relation","to_id"}}
        };
        auto columns = query("SELECT table_name,column_name,data_type,CAST(is_nullable AS VARCHAR),COALESCE(column_default,'NULL') FROM duckdb_columns() WHERE schema_name='main' AND NOT internal ORDER BY table_name,column_index LIMIT 65");
        std::size_t index=0;
        for (const auto& [table,names] : expected) for (const auto& name : names) { require(index < columns->size() && columns->string(index,0) == table && columns->string(index,1) == name && columns->string(index,2) == "VARCHAR" && columns->string(index,3) == "false" && columns->string(index,4) == "NULL","database schema columns differ"); ++index; }
        require(index == columns->size(),"unexpected database schema columns");
        auto indexes = query("SELECT index_name,table_name,expressions,CAST(is_unique AS VARCHAR) FROM duckdb_indexes() WHERE schema_name='main' ORDER BY index_name LIMIT 9");
        const std::map<std::string,std::pair<std::string,std::string>> wanted = {{"claims_predicate",{"claims","[predicate]"}},{"claims_scope",{"claims","['\"scope\"']"}},{"claims_subject",{"claims","[subject]"}},{"edges_from",{"edges","[from_id]"}},{"edges_relation",{"edges","[relation]"}},{"edges_to",{"edges","[to_id]"}},{"nodes_capture",{"nodes","[capture_digest]"}},{"nodes_kind",{"nodes","[kind]"}}};
        require(indexes->size() == wanted.size(),"database index inventory differs"); std::size_t ix=0;
        for (const auto& [name,attributes] : wanted) { require(indexes->string(ix,0)==name && indexes->string(ix,1)==attributes.first && indexes->string(ix,2)==attributes.second && indexes->string(ix,3)=="false","database index identity differs"); ++ix; }
        auto primary=query("SELECT table_name,CAST(constraint_column_names AS VARCHAR) FROM duckdb_constraints() WHERE schema_name='main' AND constraint_type='PRIMARY KEY' ORDER BY table_name LIMIT 7");
        const std::map<std::string,std::string> primary_expected={{"claims","[tops_id, namespace, snapshot_digest, row_key]"},{"edges","[tops_id, namespace, snapshot_digest, row_key]"},{"intents","[tops_id, namespace, operation_id]"},{"metadata","[key]"},{"nodes","[tops_id, namespace, snapshot_digest, row_key]"},{"snapshots","[tops_id, namespace, snapshot_digest]"}};
        require(primary->size()==primary_expected.size(),"database primary-key inventory differs"); ix=0;
        for (const auto& [table,names]:primary_expected) { require(primary->string(ix,0)==table && primary->string(ix,1)==names,"database primary-key definition differs"); ++ix; }
        auto views = query("SELECT view_name FROM duckdb_views() WHERE NOT internal LIMIT 1"); require(views->size()==0,"unexpected database view");
    }
};
std::vector<std::string> identity(const Json& value) { return {text(value.at("tops_id")),text(value.at("namespace"))}; }
Json parsed(const std::string& value) { auto out = engine::parse_bounded_json(value,engine::Limits::max_response_bytes); require(out.dump()==value,"stored JSON is not canonical"); return out; }
Json get_intent(Database& db,const Json& input,std::string& state) {
    auto parameters = identity(input); parameters.push_back(operation_id(input.at("operation_id")));
    auto row=db.query("SELECT document,intent_digest,snapshot_digest,state FROM intents WHERE tops_id=? AND namespace=? AND operation_id=? LIMIT 2",parameters);
    require(row->size()==1,"operation is absent or duplicated"); auto intent=parsed(row->string(0,0)); validate_intent(intent);
    require(intent.at("digest")==row->string(0,1) && intent.at("snapshot").at("digest")==row->string(0,2) && intent.at("operation_id")==input.at("operation_id") && intent.at("snapshot").at("tops_id")==input.at("tops_id") && intent.at("snapshot").at("namespace")==input.at("namespace"),"stored intent identity differs");
    state=row->string(0,3); require(state=="prepared" || state=="committed","stored intent state differs"); return intent;
}
Json get_snapshot(Database& db,const Json& input) {
    auto parameters=identity(input); parameters.push_back(digest(input.at("snapshot_digest")));
    auto row=db.query("SELECT document FROM snapshots WHERE tops_id=? AND namespace=? AND snapshot_digest=? LIMIT 2",parameters);
    require(row->size()==1,"snapshot is absent or duplicated"); auto snapshot=parsed(row->string(0,0)); validate_snapshot(snapshot);
    require(snapshot.at("digest")==input.at("snapshot_digest") && snapshot.at("tops_id")==input.at("tops_id") && snapshot.at("namespace")==input.at("namespace"),"stored snapshot identity differs"); return snapshot;
}
void verify_rows(Database& db,const Json& snapshot,const Json& projected) {
    auto parameters=identity(snapshot); parameters.push_back(text(snapshot.at("digest")));
    for (const auto& [kind,filter_names] : indexed_fields) {
        std::string sql="SELECT row_key,value"; for(const auto& name:filter_names) sql+=","+sql_column(name);
        sql+=" FROM "+kind+" WHERE tops_id=? AND namespace=? AND snapshot_digest=? ORDER BY row_key LIMIT "+std::to_string(projected.at(kind).size()+1);
        auto rows=db.query(sql,parameters); require(rows->size()==projected.at(kind).size(),"stored projection count differs");
        for(std::size_t i=0;i<rows->size();++i) {
            const auto& expected=projected.at(kind).at(i); require(rows->string(i,0)==text(expected.at("key")) && rows->string(i,1)==expected.at("value").dump(),"stored projection row differs");
            for(std::size_t f=0;f<filter_names.size();++f) { const auto& name=filter_names[f]; auto wanted=name=="scope"?expected.at("value").at(name).dump():text(expected.at("value").at(name)); require(rows->string(i,f+2)==wanted,"stored indexed field differs"); }
        }
    }
}
Json status_result(Database& db,const Json& intent,const std::string& state) {
    const auto& snapshot=intent.at("snapshot"); auto projected=projection(snapshot.at("graph"));
    if(state=="committed") { auto actual=get_snapshot(db,Json{{"tops_id",snapshot.at("tops_id")},{"namespace",snapshot.at("namespace")},{"snapshot_digest",snapshot.at("digest")}}); require(actual==snapshot,"committed snapshot differs"); verify_rows(db,snapshot,projected); }
    return seal(Json{{"protocol","symphony.scv.graph-index-status.v1"},{"backend","duckdb"},{"intent",intent},{"state",state},{"snapshot_digest",snapshot.at("digest")},{"projection_digest",engine::tagged_sha256(projected.dump())},{"counts",counts(projected)},{"index_verified",state=="committed"}});
}
Json prepare(const engine::Request& request) {
    const auto& input=request.payload; fields(input,{"tops_id","namespace","operation_id","graph","owner","connector","query_time"}); scope(input); static_cast<void>(operation_id(input.at("operation_id")));
    Json snapshot=seal(Json{{"protocol","symphony.scv.graph-index-snapshot.v1"},{"backend","duckdb"},{"mapping_version","1"},{"tops_id",input.at("tops_id")},{"namespace",input.at("namespace")},{"graph",input.at("graph")},{"owner",input.at("owner")},{"connector",input.at("connector")}});
    Json intent=seal(Json{{"protocol","symphony.scv.graph-index-intent.v1"},{"operation_id",input.at("operation_id")},{"snapshot",snapshot},{"validation_query_time",input.at("query_time")}}); validate_intent(intent);
    Database db(request,true); db.begin(); auto parameters=identity(input); parameters.push_back(text(input.at("operation_id")));
    auto existing=db.query("SELECT operation_id FROM intents WHERE tops_id=? AND namespace=? AND operation_id=? LIMIT 2",parameters);
    if(existing->size()!=0) { std::string state; auto saved=get_intent(db,input,state); require(saved==intent,"operation identifier conflict"); auto result=status_result(db,saved,state); db.commit(); return result; }
    auto total=db.query("SELECT operation_id FROM intents LIMIT 129"); require(total->size()<128,"intent capacity exhausted");
    parameters.push_back(text(intent.at("digest"))); parameters.push_back(text(snapshot.at("digest"))); parameters.push_back("prepared"); parameters.push_back(intent.dump());
    db.exec("INSERT INTO intents VALUES (?,?,?,?,?,?,?)",parameters); auto result=status_result(db,intent,"prepared"); db.commit(); return result;
}
Json commit(const engine::Request& request) {
    const auto& input=request.payload; fields(input,{"tops_id","namespace","operation_id","expected_intent_digest"}); scope(input); static_cast<void>(digest(input.at("expected_intent_digest")));
    Database db(request,false); db.begin(); std::string state; auto intent=get_intent(db,input,state); require(intent.at("digest")==input.at("expected_intent_digest"),"expected intent digest differs");
    if(state=="committed") { auto result=status_result(db,intent,state); db.commit(); return result; }
    const auto& snapshot=intent.at("snapshot"); auto parameters=identity(snapshot); parameters.push_back(text(snapshot.at("digest")));
    auto existing=db.query("SELECT document FROM snapshots WHERE tops_id=? AND namespace=? AND snapshot_digest=? LIMIT 2",parameters);
    auto projected=projection(snapshot.at("graph"));
    if(existing->size()) { require(existing->size()==1 && parsed(existing->string(0,0))==snapshot,"snapshot identity collision"); verify_rows(db,snapshot,projected); }
    else {
        auto total=db.query("SELECT snapshot_digest FROM snapshots LIMIT 129"); require(total->size()<128,"snapshot capacity exhausted");
        auto snapshot_parameters=parameters; snapshot_parameters.push_back(snapshot.dump()); db.exec("INSERT INTO snapshots VALUES (?,?,?,?)",snapshot_parameters);
        for(const auto& [kind,filter_names]:indexed_fields) for(const auto& row:projected.at(kind)) {
            deadline(request); auto values=parameters; values.push_back(text(row.at("key"))); values.push_back(row.at("value").dump());
            std::string sql="INSERT INTO "+kind+" VALUES (?,?,?,?,?";
            for(const auto& name:filter_names) { sql+=",?"; values.push_back(name=="scope"?row.at("value").at(name).dump():text(row.at("value").at(name))); }
            sql+=")"; db.exec(sql,values);
        }
    }
    auto operation=identity(input); operation.push_back(text(input.at("operation_id"))); db.exec("UPDATE intents SET state='committed' WHERE tops_id=? AND namespace=? AND operation_id=?",operation);
    auto result=status_result(db,intent,"committed"); db.commit(); return result;
}
Json status(const engine::Request& request) { const auto& input=request.payload; fields(input,{"tops_id","namespace","operation_id"}); scope(input); Database db(request,false); db.begin(); std::string state; auto intent=get_intent(db,input,state); auto result=status_result(db,intent,state); db.commit(); return result; }
Json exported(const engine::Request& request) {
    const auto& input=request.payload; fields(input,{"tops_id","namespace","snapshot_digest"}); scope(input); Database db(request,false); db.begin(); auto snapshot=get_snapshot(db,input); auto projected=projection(snapshot.at("graph")); verify_rows(db,snapshot,projected);
    auto result=seal(Json{{"protocol","symphony.scv.graph-index-export.v1"},{"backend","duckdb"},{"snapshot",snapshot},{"projection_digest",engine::tagged_sha256(projected.dump())},{"counts",counts(projected)}}); db.commit(); return result;
}
Json queried(const engine::Request& request) {
    const auto& input=request.payload; fields(input,{"tops_id","namespace","snapshot_digest","kind","filters","cursor","limit"}); scope(input); auto kind=text(input.at("kind")); require(indexed_fields.contains(kind),"query kind differs");
    const auto& filter_names=indexed_fields.at(kind); fields(input.at("filters"),{},std::set<std::string>(filter_names.begin(),filter_names.end()));
    for(const auto& [key,value]:input.at("filters").items()) {
        if(key!="scope") static_cast<void>(text(value,65536));
        else scope_filter(value);
    }
    require(input.at("limit").is_number_integer() && input.at("limit").get<std::int64_t>()>=1 && input.at("limit").get<std::int64_t>()<=128,"query limit outside bound"); auto limit=input.at("limit").get<std::size_t>();
    Json query_identity=input; query_identity.erase("cursor"); query_identity.erase("limit"); auto query_digest=engine::tagged_sha256(query_identity.dump());
    const auto& cursor=input.at("cursor"); if(!cursor.is_null()) { fields(cursor,{"query_digest","after_key"}); require(digest(cursor.at("query_digest"))==query_digest,"cursor query differs"); static_cast<void>(text(cursor.at("after_key"))); }
    Database db(request,false); db.begin(); auto snapshot=get_snapshot(db,input); auto projected=projection(snapshot.at("graph")); verify_rows(db,snapshot,projected);
    auto parameters=identity(input); parameters.push_back(text(input.at("snapshot_digest"))); std::string where=" WHERE tops_id=? AND namespace=? AND snapshot_digest=?";
    for(const auto& [key,value]:input.at("filters").items()) { where+=" AND "+sql_column(key)+"=?"; parameters.push_back(key=="scope"?value.dump():text(value,65536)); }
    auto matches=db.query("SELECT row_key FROM "+kind+where+" ORDER BY row_key LIMIT 1025",parameters); const auto matched_count=matches->size();
    if(!cursor.is_null()) {
        bool found=false; for(std::size_t i=0;i<matched_count;++i) if(matches->string(i,0)==text(cursor.at("after_key"))) found=true;
        require(found,"cursor key is not a matching row"); where+=" AND row_key>?"; parameters.push_back(text(cursor.at("after_key")));
    }
    auto selected=db.query("SELECT row_key,value FROM "+kind+where+" ORDER BY row_key LIMIT "+std::to_string(limit+1),parameters); Json rows=Json::array();
    for(std::size_t i=0;i<std::min(limit,selected->size());++i) rows.push_back({{"key",selected->string(i,0)},{"value",parsed(selected->string(i,1))}});
    Json next=nullptr; if(selected->size()>limit) next={{"query_digest",query_digest},{"after_key",rows.back().at("key")}};
    auto result=seal(Json{{"protocol","symphony.scv.graph-index-query.v1"},{"backend","duckdb"},{"input",input},{"snapshot",snapshot},{"projection_digest",engine::tagged_sha256(projected.dump())},{"counts",counts(projected)},{"rows",rows},{"matched_count",matched_count},{"next_cursor",next}}); db.commit(); return result;
}
}
Json descriptor() {
    std::vector<engine::OperationSpec> operations;
    for(const auto* name:{"inspect","prepare","commit","status","query","export"}) {
        std::string output=name==std::string("inspect")?engine::descriptor_protocol_v2:name==std::string("query")?"symphony.scv.graph-index-query.v1":name==std::string("export")?"symphony.scv.graph-index-export.v1":"symphony.scv.graph-index-status.v1";
        operations.push_back({std::string("engop:symphony:scv.graph-index.")+name,name,"implemented",false,true,{"ssfv:symphony:scv-graph-duckdb-connector"},{name==std::string("inspect")?"inspect":name==std::string("prepare")?"invoke":name==std::string("commit")?"invoke":"query"},"qxctl_required",std::string("symphony.scv.graph-index-")+name+"-input.v1",output,name==std::string("inspect")?"read_only":"evidence_only","idempotent",false,"none","","supported","freezing"});
    }
    for (auto& op : operations) { if (op.operation_name == "commit") { op.administrative_interactions = {"invoke","recover"}; op.expected_state_required = true; } if (op.operation_name == "status") op.administrative_interactions = {"inspect","recover"}; }
    engine::validate_operation_specs(operations);
    return seal(Json{{"protocol",engine::descriptor_protocol_v2},{"format_version",2},{"module_id","scv-graph-duckdb-connector"},{"engine_id",engine_id},{"vector_id","scv"},{"engine_version",version},{"process_protocols",Json::array({engine::process_protocol_v1})},{"contract_versions",Json::array({"knowledge/SPEC.md@v1","scv-graph-duckdb-connector/SPEC.md@v1","scv-graph-duckdb-connector/DUCKDB-PROVENANCE.json@1.5.5"})},{"operations",engine::administration_operation_descriptors(operations)},{"limits",{{"request_bytes",engine::Limits::max_request_bytes},{"response_bytes",engine::Limits::max_response_bytes},{"json_depth",engine::Limits::max_json_depth},{"json_values",engine::Limits::max_json_values},{"path_bytes",engine::Limits::max_path_bytes},{"snapshot_files",engine::Limits::max_snapshot_files},{"snapshot_file_bytes",engine::Limits::max_snapshot_file_bytes},{"deadline_ahead_ms",engine::Limits::max_deadline_ahead_ms}}},{"supported_scopes",Json::array({"tops"})},{"language","C++26"},{"thermal_path","freezing"},{"canonical_apply_enabled",false},{"session_mutation_enabled",false},{"network_listener",false}},"descriptor_digest");
}
Json handle(const engine::Request& request) {
    deadline(request);
    if(request.operation=="inspect") { fields(request.payload,{}); return descriptor(); }
    if(request.operation=="prepare") return prepare(request);
    if(request.operation=="commit") return commit(request);
    if(request.operation=="status") return status(request);
    if(request.operation=="query") return queried(request);
    if(request.operation=="export") return exported(request);
    fail("operation","unsupported connector operation");
}
} // namespace symphony::scv::duckdb_connector
