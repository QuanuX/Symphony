// Offline public-SDK qualification. No live provider or product runtime dependency.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

type object = map[string]any

const engineID = "beta-empirical-price-example"
const startTime int64 = 1609160400000000000

var tradePrices = []int64{100, 95, 105, 96, 104, 97, 103, 98, 102, 99, 101, 100}

func canonical(value any) []byte {
	validateStringEncoding(reflect.ValueOf(value))
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	must(encoder.Encode(value))
	data := bytes.TrimSuffix(out.Bytes(), []byte{'\n'})
	// The receipt owner emits these valid Unicode characters literally. Go's
	// JSON encoder escapes them even with HTML escaping disabled. Preserve
	// literal backslash-u text by consuming other escapes as complete pairs.
	result := make([]byte, 0, len(data))
	for i := 0; i < len(data); {
		if data[i] == '\\' && i+1 < len(data) {
			if bytes.HasPrefix(data[i:], []byte(`\u2028`)) {
				result = append(result, []byte("\u2028")...)
				i += 6
				continue
			}
			if bytes.HasPrefix(data[i:], []byte(`\u2029`)) {
				result = append(result, []byte("\u2029")...)
				i += 6
				continue
			}
			result = append(result, data[i:i+2]...)
			i += 2
			continue
		}
		result = append(result, data[i])
		i++
	}
	return result
}
func validateStringEncoding(value reflect.Value) {
	if !value.IsValid() {
		return
	}
	switch value.Kind() {
	case reflect.String:
		require(utf8.ValidString(value.String()), "invalid UTF-8 JSON string")
	case reflect.Interface, reflect.Pointer:
		if !value.IsNil() {
			validateStringEncoding(value.Elem())
		}
	case reflect.Map:
		for _, key := range value.MapKeys() {
			validateStringEncoding(key)
			validateStringEncoding(value.MapIndex(key))
		}
	case reflect.Array, reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			validateStringEncoding(value.Index(i))
		}
	}
}
func validateUnicodeEscapes(data []byte) {
	inString := false
	for i := 0; i < len(data); i++ {
		if data[i] == '"' {
			inString = !inString
			continue
		}
		if !inString || data[i] != '\\' || i+1 >= len(data) {
			continue
		}
		if data[i+1] != 'u' || i+6 > len(data) {
			i++
			continue
		}
		code, err := strconv.ParseUint(string(data[i+2:i+6]), 16, 16)
		if err != nil {
			continue // The JSON decoder rejects malformed escapes below.
		}
		if code >= 0xd800 && code <= 0xdbff {
			require(i+12 <= len(data) && data[i+6] == '\\' && data[i+7] == 'u', "unpaired JSON surrogate")
			low, err := strconv.ParseUint(string(data[i+8:i+12]), 16, 16)
			require(err == nil && low >= 0xdc00 && low <= 0xdfff, "unpaired JSON surrogate")
			i += 11
		} else {
			require(code < 0xdc00 || code > 0xdfff, "unpaired JSON surrogate")
			i += 5
		}
	}
}
func decode(data []byte) any {
	require(utf8.Valid(data), "invalid UTF-8 JSON input")
	validateUnicodeEscapes(data)
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	must(decoder.Decode(&value))
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		panic("trailing JSON data")
	}
	return value
}
func readJSON(path string) any { return decode(read(path)) }
func read(path string) []byte  { data, err := os.ReadFile(path); must(err); return data }
func obj(value any) object {
	valueMap, ok := value.(map[string]any)
	if !ok {
		panic("JSON object required")
	}
	return valueMap
}
func array(value any) []any {
	values, ok := value.([]any)
	if !ok {
		panic("JSON array required")
	}
	return values
}
func str(value any) string {
	text, ok := value.(string)
	if !ok {
		panic("JSON string required")
	}
	return text
}
func clone(value object) object { return obj(decode(canonical(value))) }
func digestBytes(data []byte) string {
	result := sha256.Sum256(data)
	return hex.EncodeToString(result[:])
}
func digest(path string) string { return digestBytes(read(path)) }
func equal(a, b any) bool       { return bytes.Equal(canonical(a), canonical(b)) }
func must(err error) {
	if err != nil {
		panic(err)
	}
}
func require(condition bool, message string) {
	if !condition {
		panic(message)
	}
}
func exists(path string) bool {
	_, err := os.Lstat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	panic(err)
}
func inside(path, parent string) bool {
	relative, err := filepath.Rel(parent, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}
func copyFile(from, to string, mode fs.FileMode) {
	data := read(from)
	must(os.MkdirAll(filepath.Dir(to), 0700))
	output, err := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode.Perm())
	must(err)
	_, err = output.Write(data)
	closeErr := output.Close()
	must(err)
	must(closeErr)
}
func copyTree(from, to string) {
	must(filepath.WalkDir(from, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		target := filepath.Join(to, relative)
		if entry.IsDir() {
			return os.Mkdir(target, 0700)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinked SDK copy member: %s", relative)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return errors.New("regular SDK copy member required")
		}
		copyFile(path, target, info.Mode())
		return nil
	}))
}
func rational(value *big.Rat) object {
	return object{"numerator": value.Num().String(), "denominator": value.Denom().String()}
}

// The oracle accumulates exact probability mass and prices with arbitrary-
// precision rationals, independently of the example's bounded int64 arithmetic.
func oracle(prices []int64, version string) object {
	require(len(prices) > 0, "oracle requires observations")
	support := map[int64]*big.Rat{}
	unit := new(big.Rat).SetFrac(big.NewInt(1), big.NewInt(int64(len(prices))))
	total := new(big.Int)
	for _, price := range prices {
		if support[price] == nil {
			support[price] = new(big.Rat)
		}
		support[price].Add(support[price], unit)
		total.Add(total, big.NewInt(price))
	}
	keys := make([]int64, 0, len(support))
	for price := range support {
		keys = append(keys, price)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	rows := make([]any, 0, len(keys))
	for _, price := range keys {
		rows = append(rows, object{"price_nanos": strconv.FormatInt(price, 10), "weight": rational(support[price])})
	}
	return object{"protocol": "quanux.beta.empirical-price-result.v1", "engine_id": engineID, "engine_version": version,
		"sample_count": strconv.Itoa(len(prices)), "support": rows, "mean_price_nanos": rational(new(big.Rat).SetFrac(total, big.NewInt(int64(len(prices)))))}
}
func request(operation string, payload object, target ...string) object {
	engine := "symphony-sbv"
	if len(target) > 0 {
		engine = target[0]
	}
	return object{"protocol": "symphony.knowledge.engine-process.v2", "request_id": "beta-" + operation,
		"correlation_id": "beta-qualification", "operation": operation, "target_engine": engine, "deadline_unix_ms": nil, "payload": payload}
}
func unavailable(reason string) object {
	return object{"status": "unavailable", "reason": reason, "value": nil}
}
func outcome(expected object, index int, version, providerSHA string) object {
	ordinal := int64(3 + index*2)
	return object{"protocol": "symphony.sbv.model-outcome.v1", "signal_id": "signal-" + strconv.Itoa(index), "model_id": "native_provider", "model_version": "1", "status": "available", "reason": "",
		"support": expected["support"], "measure": "probability", "conditioning": "scenario", "calibration": "not_verified", "calibration_reference": "", "evidence_origin": "native_provider",
		"evidence_reference": "sha256:" + digestBytes(canonical(expected)), "producer": object{"id": engineID, "version": version, "artifact_sha256": providerSHA, "reproducibility": "deterministic_declared"},
		"horizon_end_ns": strconv.FormatInt(startTime+ordinal*100+200, 10), "horizon_within_observed_span": true, "price_scale": "1e-9", "price_unit": "price_nanos",
		"fill_probability": unavailable("Empirical prices do not estimate execution probability"), "no_fill_probability": unavailable("execution probability unavailable"),
		"fill_quantity": unavailable("not modeled by this contract"), "fill_time": unavailable("not modeled by this contract"), "actual_execution": unavailable("no actual execution supplied")}
}

type campaign struct {
	source, sdk, work, cxx, project, sentinel, sentinelSHA string
	env, checks, sourceFiles                               []string
	serial                                                 int
	sdkBefore                                              map[string]string
}

func newCampaign(source, sdk, work, cxx string) *campaign {
	var err error
	source, err = filepath.Abs(source)
	must(err)
	source, err = filepath.EvalSymlinks(source)
	must(err)
	sdk, err = filepath.Abs(sdk)
	must(err)
	sdk, err = filepath.EvalSymlinks(sdk)
	must(err)
	require(filepath.IsAbs(work) && !exists(work), "new absolute work directory required")
	parent, err := filepath.EvalSymlinks(filepath.Dir(work))
	must(err)
	work = filepath.Join(parent, filepath.Base(work))
	require(!inside(work, sdk) && !inside(work, source), "work must be outside SDK and source")
	// Retain clang++ argv[0], not its symlink target clang, for C++ linkage.
	cxx, err = filepath.Abs(cxx)
	must(err)
	info, err := os.Stat(cxx)
	must(err)
	require(info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0, "executable C++ driver required")
	must(os.Mkdir(work, 0700))
	for _, name := range []string{"logs", "outputs", "home", "tmp", "example-install"} {
		must(os.Mkdir(filepath.Join(work, name), 0700))
	}
	omitted := map[string]bool{}
	for _, key := range []string{"HOME", "TMPDIR", "CPATH", "CPLUS_INCLUDE_PATH", "C_INCLUDE_PATH", "LIBRARY_PATH", "CMAKE_PREFIX_PATH", "CMAKE_TOOLCHAIN_FILE", "DYLD_LIBRARY_PATH", "DYLD_FALLBACK_LIBRARY_PATH", "LD_LIBRARY_PATH"} {
		omitted[key] = true
	}
	env := []string{}
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if !omitted[key] {
			env = append(env, entry)
		}
	}
	env = append(env, "HOME="+filepath.Join(work, "home"), "TMPDIR="+filepath.Join(work, "tmp"))
	c := &campaign{source: source, sdk: sdk, work: work, cxx: cxx, env: env, checks: []string{}, project: filepath.Join(work, "source")}
	c.sdkBefore = c.verifySDK()
	must(os.Mkdir(c.project, 0700))
	c.sourceFiles = []string{"CMakeLists.txt", "calculation.hpp", "engine.cpp", "provider.cpp", "sdk-client.cpp", "provider-probe.cpp", "fixture.cpp", "qualify.go", "README.md", "fixtures/public_fixture.hpp", "fixtures/ATTRIBUTION.md", "fixtures/LICENSE.Apache-2.0"}
	for _, name := range c.sourceFiles {
		copyFile(filepath.Join(source, name), filepath.Join(c.project, name), 0600)
	}
	c.sentinel = filepath.Join(work, "example-install/user-research.txt")
	c.save(c.sentinel, []byte("User-owned unrelated research; preserve unchanged.\n"))
	c.sentinelSHA = digest(c.sentinel)
	return c
}
func (c *campaign) save(path string, value any) {
	data, ok := value.([]byte)
	if !ok {
		var err error
		data, err = json.MarshalIndent(value, "", "  ")
		must(err)
		data = append(data, '\n')
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	must(err)
	_, err = file.Write(data)
	closeErr := file.Close()
	must(err)
	must(closeErr)
}
func (c *campaign) check(condition bool, label string) {
	require(condition, label)
	c.checks = append(c.checks, label)
}
func (c *campaign) run(label string, command []string, value object, success bool) object {
	c.serial++
	stem := filepath.Join(c.work, "logs", fmt.Sprintf("%03d-%s", c.serial, label))
	if value != nil {
		c.save(stem+".input.json", value)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = c.work
	cmd.Env = c.env
	if value != nil {
		cmd.Stdin = bytes.NewReader(canonical(value))
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	c.save(stem+".stdout", stdout.Bytes())
	c.save(stem+".stderr", stderr.Bytes())
	if ctx.Err() != nil {
		panic(label + ": process timeout")
	}
	// A missing executable is not an expected product refusal.
	if err != nil {
		var exit *exec.ExitError
		require(errors.As(err, &exit), label+": command could not execute")
	}
	c.check((err == nil) == success, label+": expected process status")
	if value != nil && success {
		return obj(decode(stdout.Bytes()))
	}
	return nil
}
func (c *campaign) verifySDK() map[string]string {
	receipts := map[string]string{}
	for _, identity := range [][2]string{{"knowledge-vector-engine-cpp", "0.2.0-dev"}, {"sbv-engine", "0.33.0-dev"}} {
		component, version := identity[0], identity[1]
		relative := filepath.Join("share/symphony/receipts", component, version, "install-receipt.json")
		path := filepath.Join(c.sdk, relative)
		info, err := os.Lstat(path)
		must(err)
		require(info.Mode().IsRegular(), "complete public receipt required")
		data := obj(readJSON(path))
		expected := str(data["receipt_digest"])
		delete(data, "receipt_digest")
		require(expected == "sha256:"+digestBytes(canonical(data)), "receipt digest mismatch")
		require(data["component_id"] == component && data["version"] == version, "exact SDK identity required")
		for _, raw := range array(data["files"]) {
			entry := obj(raw)
			relative := str(entry["path"])
			require(!filepath.IsAbs(relative) && !containsParent(relative), "unsafe SDK receipt path")
			owned := filepath.Join(c.sdk, relative)
			for p := owned; p != c.sdk; p = filepath.Dir(p) {
				require(inside(p, c.sdk), "unsafe SDK receipt path")
				i, e := os.Lstat(p)
				must(e)
				require(i.Mode()&os.ModeSymlink == 0, "symlinked SDK receipt member")
			}
			info, err := os.Stat(owned)
			must(err)
			size, err := strconv.ParseInt(fmt.Sprint(entry["size"]), 10, 64)
			must(err)
			require(info.Mode().IsRegular() && info.Size() == size && "sha256:"+digest(owned) == entry["digest"], "SDK owned byte mismatch")
		}
		receipts[filepath.ToSlash(relative)] = digest(path)
	}
	return receipts
}
func containsParent(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}
func (c *campaign) build(version string) (string, map[string]string) {
	build := filepath.Join(c.work, "build-"+version)
	c.run("configure-"+version, []string{"cmake", "-S", c.project, "-B", build, "-DCMAKE_CXX_COMPILER=" + c.cxx, "-DCMAKE_BUILD_TYPE=Release", "-DBETA_SDK_PREFIX=" + c.sdk, "-DBETA_EXAMPLE_VERSION=" + version, "-DCMAKE_INSTALL_PREFIX=" + filepath.Join(c.work, "example-install")}, nil, true)
	c.run("build-"+version, []string{"cmake", "--build", build, "--parallel", "2"}, nil, true)
	c.run("install-"+version, []string{"cmake", "--install", build}, nil, true)
	commands := array(readJSON(filepath.Join(build, "compile_commands.json")))
	all := true
	for _, entry := range commands {
		all = all && inside(str(obj(entry)["file"]), c.project)
	}
	c.check(all, "independent copied source only: "+version)
	deps := []string{}
	links := []string{}
	must(filepath.WalkDir(build, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			if strings.HasSuffix(path, ".o.d") {
				deps = append(deps, path)
			}
			if entry.Name() == "link.txt" {
				links = append(links, path)
			}
		}
		return nil
	}))
	for _, path := range append(deps, links...) {
		text := string(read(path))
		relative, err := filepath.Rel(build, path)
		must(err)
		c.check(!strings.Contains(text, c.source) && !strings.Contains(text, strings.ReplaceAll(c.source, " ", "\\ ")), "no original-source dependency: "+filepath.ToSlash(relative))
	}
	installed := strings.Split(strings.TrimSpace(string(read(filepath.Join(build, "install_manifest.txt")))), "\n")
	all = len(installed) == 2
	for _, path := range installed {
		all = all && inside(path, filepath.Join(c.work, "example-install"))
	}
	c.check(all, "independent exact example install inventory: "+version)
	inventory := map[string]string{}
	for _, path := range installed {
		inventory[path] = digest(path)
	}
	c.save(filepath.Join(c.work, "outputs", "installed-"+version+".json"), inventory)
	c.run("abi-probe-"+version, []string{filepath.Join(build, "beta-provider-probe"), filepath.Join(build, "beta-empirical-provider.so")}, nil, true)
	return build, inventory
}
func (c *campaign) direct(binary string, prices []int64, version, label string) {
	stringsPrices := []any{}
	for _, p := range prices {
		stringsPrices = append(stringsPrices, strconv.FormatInt(p, 10))
	}
	response := c.run(label, []string{binary}, request("empirical_distribution", object{"protocol": "quanux.beta.empirical-price-input.v1", "prices_nanos": stringsPrices}, engineID), true)
	expected := oracle(prices, version)
	c.check(equal(response["result"], expected) && response["outcome"] == "ok", label+": complete exact oracle")
	c.save(filepath.Join(c.work, "outputs", label+"-oracle.json"), expected)
}
func removal(inventory map[string]string) error {
	paths := []string{}
	for path, sha := range inventory {
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("example removal refused changed owned bytes")
		}
		bytes, err := os.ReadFile(path)
		if err != nil || digestBytes(bytes) != sha {
			return errors.New("example removal refused changed owned bytes")
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}
func (c *campaign) interrupt(binary string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary)
	cmd.Dir = c.work
	cmd.Env = c.env
	input, err := cmd.StdinPipe()
	must(err)
	defer input.Close()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	must(cmd.Start())
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
		panic("waiting process exited before interruption")
	case <-time.After(50 * time.Millisecond):
	}
	c.check(true, "bounded interruption reached waiting process")
	must(cmd.Process.Signal(syscall.SIGTERM))
	// Wait may close the pipe concurrently after SIGTERM; closure is idempotent
	// for this fixture and must not turn a real interruption into a false failure.
	_ = input.Close()
	err = <-done
	c.save(filepath.Join(c.work, "outputs/interruption.stdout"), stdout.Bytes())
	c.check(ctx.Err() == nil && err != nil && stdout.Len() == 0, "interrupted input does not fabricate completion")
}
func (c *campaign) execute() {
	old, oldFiles := c.build("0.1.0")
	newBuild, newFiles := c.build("0.1.1")
	maximum := make([]int64, 64)
	for i := range maximum {
		maximum[i] = 1000000000000000
	}
	for _, v := range [][2]string{{"0.1.0", old}, {"0.1.1", newBuild}} {
		for i, prices := range [][]int64{{100, 95, 105, 95}, {-3, -3, 2, 4}, maximum, {7}} {
			c.direct(filepath.Join(v[1], "beta-empirical-price"), prices, v[0], v[0]+"-direct-"+strconv.Itoa(i))
		}
	}
	c.run("missing-sdk", []string{"cmake", "-S", c.project, "-B", filepath.Join(c.work, "missing-sdk"), "-DBETA_SDK_PREFIX=" + filepath.Join(c.work, "home")}, nil, false)
	wrongSDK := filepath.Join(c.work, "wrong-sdk-copy")
	copyTree(c.sdk, wrongSDK)
	cfg := filepath.Join(wrongSDK, "lib/cmake/SymphonySbvProvider/0.33.0-dev/SymphonySbvProviderConfig.cmake")
	must(os.WriteFile(cfg, []byte(strings.ReplaceAll(string(read(cfg)), `set(SymphonySbvProvider_VERSION "0.33.0-dev")`, `set(SymphonySbvProvider_VERSION "0.32.0-dev")`)), 0600))
	c.run("wrong-sdk", []string{"cmake", "-S", c.project, "-B", filepath.Join(c.work, "wrong-sdk-build"), "-DBETA_SDK_PREFIX=" + wrongSDK}, nil, false)
	binary := filepath.Join(old, "beta-empirical-price")
	valid := request("empirical_distribution", object{"protocol": "quanux.beta.empirical-price-input.v1", "prices_nanos": []any{"1"}}, engineID)
	oversized := make([]any, 65)
	for i := range oversized {
		oversized[i] = "1"
	}
	for _, test := range []struct {
		label  string
		prices []any
	}{{"empty", []any{}}, {"oversize", oversized}, {"overflow", []any{"1000000000000001"}}, {"noncanonical", []any{"01"}}, {"numeric", []any{1}}} {
		bad := clone(valid)
		obj(bad["payload"])["prices_nanos"] = test.prices
		c.run("direct-refuse-"+test.label, []string{binary}, bad, false)
	}
	for _, label := range []string{"schema", "extra", "target", "deadline"} {
		bad := clone(valid)
		switch label {
		case "schema":
			obj(bad["payload"])["protocol"] = "other"
		case "extra":
			obj(bad["payload"])["extra"] = true
		case "target":
			bad["target_engine"] = "other"
		case "deadline":
			bad["deadline_unix_ms"] = "1"
		}
		c.run("direct-refuse-"+label, []string{binary}, bad, false)
	}
	c.interrupt(binary)
	source := filepath.Join(c.work, "outputs/synthetic.dbn")
	c.run("synthetic-dbn", []string{filepath.Join(old, "beta-fixture"), source}, nil, true)
	templates := obj(obj(readJSON(filepath.Join(c.sdk, "share/symphony/schemas/sbv-engine/0.33.0-dev/admin.templates.json")))["templates"])
	run := clone(obj(templates["run"]))
	run["source_path"] = source
	run["source_sha256"] = digest(source)
	run["output_path"] = filepath.Join(c.work, "outputs/census.json")
	run["criteria"] = object{"rule": "spaced_trades", "spacing_ns": "200", "min_trade_size": "1", "direction": "any", "max_signals": "4"}
	run["execution"] = object{"model": "none", "horizon_ns": "200", "price_offsets_nanos": []any{}, "probability_numerator": "0", "probability_denominator": "1"}
	run["replay"] = object{"before_ns": "1000", "after_ns": "200", "retain_events": true}
	run["studies"] = []any{"signal_summary"}
	run["workers"] = "1"
	c.run("sdk-census", []string{filepath.Join(old, "beta-sdk-client")}, request("run", run), true)
	census := obj(readJSON(str(run["output_path"])))
	ordinals := []string{}
	for _, signal := range array(obj(obj(census["sections"])["signals"])["data"]) {
		ordinals = append(ordinals, str(obj(signal)["source_ordinal"]))
	}
	c.check(reflect.DeepEqual(ordinals, []string{"3", "5", "7", "9"}), "independent census order oracle")
	for _, v := range [][2]string{{"0.1.0", old}, {"0.1.1", newBuild}} {
		version, build := v[0], v[1]
		library := filepath.Join(build, "beta-empirical-provider.so")
		provider := object{"protocol": "symphony.sbv.native-provider-selection.v1", "library": object{"path": library, "expected_sha256": digest(library)}, "id": engineID, "version": version, "role": "model", "input_profile": "symphony.sbv.provider-databento-mbo-event.v1", "concurrency": "serialized_instance", "parameters": object{"version": "1", "window": "4"}, "dependencies": object{"capture": "uncaptured", "description": "Exact installed public SDK example and platform runtime", "artifacts": []any{}}, "extensions": object{}}
		evaluation := clone(obj(templates["evaluate"]))
		evaluation["source_path"] = source
		evaluation["source_sha256"] = digest(source)
		evaluation["output_path"] = filepath.Join(c.work, "outputs", "evaluation-"+version+".json")
		evaluation["census"] = object{"path": run["output_path"], "expected_sha256": census["content_sha256"], "pointer": ""}
		evaluation["model"] = object{"protocol": "symphony.sbv.model-selection.v1", "id": "native_provider", "version": "1", "horizon_ns": "200", "parameters": object{"provider": provider, "measure": "probability", "conditioning": "scenario", "calibration_reference": "", "on_unavailable": "reject"}}
		evaluation["replay"] = run["replay"]
		evaluation["studies"] = []any{"signal_summary", "model_summary"}
		evaluation["workers"] = "1"
		c.run("sdk-evaluation-"+version, []string{filepath.Join(build, "beta-sdk-client")}, request("evaluate", evaluation), true)
		result := obj(readJSON(str(evaluation["output_path"])))
		expected := []any{}
		for i := 0; i < 4; i++ {
			through := i*2 + 1
			from := through - 4
			if from < 0 {
				from = 0
			}
			prices := []int64{}
			for _, p := range tradePrices[from:through] {
				prices = append(prices, p*1000000000)
			}
			computed := oracle(prices, version)
			c.direct(filepath.Join(build, "beta-empirical-price"), prices, version, version+"-causal-"+strconv.Itoa(i))
			expected = append(expected, outcome(computed, i, version, digest(library)))
		}
		c.save(filepath.Join(c.work, "outputs", "full-execution-oracle-"+version+".json"), expected)
		c.check(equal(obj(obj(result["sections"])["execution"])["data"], expected), "complete native-provider outcome oracle: "+version)
		inspected := obj(c.run("complete-inspect-"+version, []string{filepath.Join(build, "beta-sdk-client")}, request("result_inspect", object{"protocol": "symphony.sbv.result-inspect-input.v1", "path": evaluation["output_path"], "expected_sha256": result["content_sha256"]}), true)["result"])
		exported := read(str(evaluation["output_path"]))
		c.check(inspected["expected_digest_verified"] == true && inspected["file_sha256"] == digestBytes(exported) && inspected["bytes"] == strconv.Itoa(len(exported)) && equal(decode(exported), result), "complete native-validated result read: "+version)
		before := digest(str(evaluation["output_path"]))
		// This retains a committed result and refuses overwrite; it is not a real
		// injected transport failure, nor permission to retry uncertain mutations.
		c.run("lost-reply-no-overwrite-"+version, []string{filepath.Join(build, "beta-sdk-client")}, request("evaluate", evaluation), false)
		c.check(digest(str(evaluation["output_path"])) == before, "lost-reply retained bytes unchanged: "+version)
		for _, label := range []string{"library-digest", "identity", "config-version", "source-digest", "deadline"} {
			bad := clone(evaluation)
			output := filepath.Join(c.work, "outputs", version+"-refuse-"+label+".json")
			bad["output_path"] = output
			selected := obj(obj(obj(bad["model"])["parameters"])["provider"])
			switch label {
			case "library-digest":
				obj(selected["library"])["expected_sha256"] = strings.Repeat("0", 64)
			case "identity":
				selected["version"] = "9.9.9"
			case "config-version":
				obj(selected["parameters"])["version"] = "2"
			case "source-digest":
				bad["source_sha256"] = strings.Repeat("0", 64)
			}
			message := request("evaluate", bad)
			if label == "deadline" {
				message["deadline_unix_ms"] = "1"
			}
			c.run(version+"-refuse-"+label, []string{filepath.Join(build, "beta-sdk-client")}, message, false)
			c.check(!exists(output), "refused publication: "+version+":"+label)
		}
	}
	paths := []string{}
	for path := range newFiles {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	changed := paths[0]
	prior := read(changed)
	must(os.WriteFile(changed, append(append([]byte{}, prior...), []byte("tamper")...), 0600))
	err := removal(newFiles)
	require(err != nil, "changed owned bytes removed")
	all := true
	for path := range newFiles {
		all = all && exists(path)
	}
	c.check(all, "removal refusal preflights entire ownership set")
	must(os.WriteFile(changed, prior, 0600))
	must(removal(newFiles))
	oldBinary := filepath.Join(c.work, "example-install/bin/beta-empirical-price/0.1.0/beta-empirical-price")
	c.direct(oldBinary, []int64{1, 3, 4}, "0.1.0", "compatible-rollback")
	c.check(digest(c.sentinel) == c.sentinelSHA, "upgrade and rollback preserve unrelated user file")
	must(removal(oldFiles))
	all = true
	for path := range oldFiles {
		all = all && !exists(path)
	}
	c.check(digest(c.sentinel) == c.sentinelSHA && all, "removal preserves unrelated user file")
	c.check(reflect.DeepEqual(c.verifySDK(), c.sdkBefore), "original complete SDK receipts and bytes unchanged")
	if runtime.GOOS == "darwin" {
		c.run("loader-closure", []string{"otool", "-L", filepath.Join(old, "beta-empirical-provider.so"), filepath.Join(old, "beta-sdk-client")}, nil, true)
	} else {
		c.run("loader-closure", []string{"ldd", filepath.Join(old, "beta-empirical-provider.so")}, nil, true)
	}
	sourceHashes := map[string]string{}
	for _, name := range c.sourceFiles {
		sourceHashes[name] = digest(filepath.Join(c.project, name))
	}
	evidenceHashes := map[string]string{}
	for _, folder := range []string{"logs", "outputs"} {
		entries, err := os.ReadDir(filepath.Join(c.work, folder))
		must(err)
		for _, entry := range entries {
			if entry.Type().IsRegular() {
				path := filepath.Join(c.work, folder, entry.Name())
				evidenceHashes[filepath.ToSlash(filepath.Join(folder, entry.Name()))] = digest(path)
			}
		}
	}
	c.save(filepath.Join(c.work, "evidence.json"), object{"protocol": "symphony.public-beta.external-engine-campaign.v1", "scope": "private_installed_sdk_fixture", "engine_id": engineID, "versions": []string{"0.1.0", "0.1.1"}, "checks": c.checks, "check_count": len(c.checks), "sdk_receipts": c.sdkBefore, "example_source_sha256": sourceHashes, "evidence_files_sha256": evidenceHashes, "complete_exact_math_oracle": true, "final_download_qualified": false, "independent_author_review": false, "documentation_only_agent_attempt": "pending independently assigned reviewer", "administrative_stall_experiment": "prepared in README; not executed", "release_acceptance_complete": false, "publication_authorized": false})
	fmt.Println(string(canonical(object{"result": "passed_private_installed_sdk_fixture", "checks": len(c.checks), "release_acceptance_complete": false})))
}
func main() {
	defer func() {
		if failure := recover(); failure != nil {
			fmt.Fprintln(os.Stderr, "external SDK qualification failed:", failure)
			os.Exit(1)
		}
	}()
	_, filename, _, ok := runtime.Caller(0)
	require(ok, "qualification source location unavailable")
	source := flag.String("source", filepath.Dir(filename), "public example source directory")
	sdk := flag.String("sdk-prefix", "", "complete installed public SDK prefix")
	work := flag.String("work", "", "new absolute private work directory")
	cxx := flag.String("cxx", "", "absolute C++ driver path; preserve clang++ name")
	flag.Parse()
	require(flag.NArg() == 0 && *sdk != "" && *work != "" && *cxx != "", "--sdk-prefix, --work and --cxx required")
	newCampaign(*source, *sdk, *work, *cxx).execute()
}
