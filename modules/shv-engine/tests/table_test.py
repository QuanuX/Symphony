"""Process-level expectations for explicit table profiles and unaltered row context."""
import copy
import hashlib
import json
import subprocess
import sys
import tempfile
import time
from pathlib import Path
sys.dont_write_bytecode = True
import kernel_test as kernel

calls = 0
negative = 0


def call(op, payload, good=True):
    global calls, negative
    calls += 1
    negative += not good
    request = dict(protocol='symphony.knowledge.engine-process.v1', request_id='tables',
                   correlation_id='tables', operation=op, target_engine='symphony-shv',
                   deadline_unix_ms=int(time.time()*1000)+30000, payload=payload)
    response = subprocess.run([kernel.args.engine], input=kernel.canonical(request), capture_output=True)
    assert not response.stderr, response.stderr
    result = json.loads(response.stdout)
    assert result == kernel.seal(result, 'response_digest')
    assert (response.returncode == 0) == good, result
    assert result['outcome'] == ('ok' if good else 'error')
    return result.get('result')


def test_table_interpretation():
    head = '<div id="heading"><h1>Example CPU®</h1></div>'
    scalar = '<section id="spec"><table><tr><th>Cores</th><td>24</td></tr><tr><th>Launch</th><td>Q1\'23</td></tr><tr><th>End</th><td>done</td></tr></table></section>'
    matrix = '<section id="modes"><table><tr><th>Config</th><th>Cores</th><th>Frequency</th><th>TDP</th><th>Description</th></tr><tr><td>CPU(0)</td><td>24</td><td>2.0</td><td>185</td><td></td></tr><tr><td>CPU(1)</td><td>16</td><td>2.3</td><td>165</td><td></td></tr><tr><td>CPU(2)</td><td>12</td><td>2.7</td><td>165</td><td></td></tr></table></section>'
    html = head + scalar + matrix
    spec = dict(id='cpu', manufacturer='Example', model='Example CPU®', hardware_class='cpu',
                source_id='s', heading_section='div#heading', interpretation_profile='scoped_tables.v1',
                fields=[dict(predicate='cores', section='section#spec', label='Cores', next_label='Launch', value_type='integer', qualifier='documented'),
                        dict(predicate='model_introduction', section='section#spec', label='Launch', next_label='End', value_type='quarter_20yy', qualifier='documented'),
                        dict(predicate='profiles', section='section#modes', columns=['Config','Cores','Frequency','TDP','Description'], value_type='table_rows', qualifier='source_rows_units_unspecified')])
    with tempfile.TemporaryDirectory(prefix='shv-tables-') as tmp:
        root = Path(tmp).resolve()

        def build(raw=html, mutate=None, good=True):
            raw = raw if isinstance(raw, bytes) else raw.encode()
            (root/'s.html').write_bytes(raw)
            payload = dict(source_root=str(root), sources=[dict(id='s', path='s.html', bytes=len(raw),
                         digest='sha256:'+hashlib.sha256(raw).hexdigest(), format='html')], subjects=[copy.deepcopy(spec)])
            if mutate:
                mutate(payload)
            return call('catalogue_build', payload, good)

        cat = build()
        values = {a['predicate']: a['value'] for a in cat['subjects'][0]['assertions']}
        assert values['cores'] == 24
        assert values['model_introduction'] == dict(precision='quarter', source_text="Q1'23", **{'from':'2023-01-01','through':'2023-03-31'})
        assert cat['subjects'][0]['introduced'] == {'from':'2023-01-01','through':'2023-03-31'}
        expected_rows = [['CPU(0)','24','2.0','185',''],['CPU(1)','16','2.3','165',''],['CPU(2)','12','2.7','165','']]
        assert values['profiles'] == dict(columns=['Config','Cores','Frequency','TDP','Description'], rows=expected_rows)
        def last_field(p):
            p['subjects'][0]['fields'][0].update(label='End',next_label=None,value_type='string')
        last = build(mutate=last_field)
        assert next(a for a in last['subjects'][0]['assertions'] if a['predicate']=='cores')['value']=='done'
        build(html.replace('</table></section>', '<tr><th>Appended</th><td>new</td></tr></table></section>',1),mutate=last_field,good=False)
        build(mutate=lambda p:p['subjects'][0]['fields'][0].update(next_label=None),good=False)
        # Dimensions exhaust visibly rather than truncating matrices.
        marker='<tr><td>CPU(2)</td><td>12</td><td>2.7</td><td>165</td><td></td></tr>'
        extra=''.join(marker.replace('CPU(2)',f'CPU({i})') for i in range(3,32))
        assert len(build(html.replace(marker,marker+extra))['subjects'][0]['assertions'][2]['value']['rows'])==32
        build(html.replace(marker,marker+extra+marker.replace('CPU(2)','CPU(32)')),good=False)
        for quarter, end in [('1','03-31'),('2','06-30'),('3','09-30'),('4','12-31')]:
            s = build(html.replace("Q1'23",f"Q{quarter}'00"))['subjects'][0]
            assert s['introduced']['through'] == '2000-'+end
        assert build(html.replace('CPU®','CPU&#174;'))['subjects'] == cat['subjects']
        assert build(html.replace('<table>','<table><tbody>').replace('</table>','</tbody></table>'))['subjects'] == cat['subjects']
        assert build('<nav><table><tr><th>Cores</th><td>99</td></tr></table></nav>'+html)['subjects'] == cat['subjects']
        assert build(html.replace('<td>24</td>', '<td><span>24</span><!-- ignored --><script>"</scriptx>99"</script></td>', 1))['subjects'] == cat['subjects']
        for old, new in [('<h1>','<h1-fake>'),('Example CPU®','Wrong CPU'),('id="spec"','id="missing"'),
                         ("Q1'23", "Q0'23"),("Q1'23", "Q5'23"),("Q1'23", "Q1’23"),("Q1'23", "Q1'2023"),
                         ("Q1'23", "Q1'2x"),('<td>24</td>','<td>024</td>'),('<th>Launch</th>','<th>Cores</th>'),
                         ('<th>End</th>','<th>Different</th>'),('<th>Config</th>','<th>Renamed</th>'),
                         ('<td>CPU(1)</td>','<td>CPU(0)</td>'),('<td>CPU(1)</td>','<td></td>'),
                         ('<td>CPU(1)</td>','<th>CPU(1)</th>'),('<td>2.3</td>',''),
                         ('<td>2.3</td>','<td colspan="1">2.3</td>'),('<td>2.3</td>','<td rowspan="2">2.3</td>'),
                         ('<td>2.3</td>','<td>2.3<td>3.0</td></td>'),('<td>2.3</td>','<td><table><tr><td>2.3</td></tr></table></td>'),
                         ('<td>2.3</td>','<td>'+'x'*4097+'</td>'),('</table></section>','</section>')]:
            build(html.replace(old,new,1), good=False)
        for old,new in [('<td>24</td>','<td>24</td> only valid in reduced mode'),
                        ('<td>24</td>','<td>24</td><div> only valid in reduced mode </div>'),
                        ('<td>24</td>','<td>24</td colspan="2">'),
                        ('<td>24</td>','<td>24'+'<b></b>'*33000+'</td>'),
                        ('<table>','<table><tbody>'),
                        ('<td>24</td>','<td><tbody>24</tbody></td>')]:
            build(html.replace(old,new,1),good=False)
        for raw in [head+head+scalar+matrix, head+scalar+scalar+matrix,
                    head+scalar+matrix.replace('</section>', '<table><tr><td>x</td></tr></table></section>'),
                    html+'\0', html.encode()+b'\xff']:
            build(raw, good=False)
        for mutate in [lambda p:p['subjects'][0].update(interpretation_profile='unknown.v1'),
                       lambda p:p['subjects'][0].update(field_section='section#spec'),
                       lambda p:p['subjects'][0]['fields'][0].update(section='div#heading'),
                       lambda p:p['subjects'][0]['fields'][2].update(columns=['Config','Config']),
                       lambda p:p['subjects'][0]['fields'][2].update(columns=[]),
                       lambda p:p['subjects'][0]['fields'][0].update(value_type='table_rows'),
                       lambda p:p['subjects'][0]['fields'][1].update(value_type='string'),
                       lambda p:p['subjects'][0]['fields'].append(copy.deepcopy(p['subjects'][0]['fields'][0]))]:
            build(mutate=mutate,good=False)
        cat = build()
        query = dict(source_root=str(root), catalogue=cat, subject_ids=['cpu'])
        assert call('catalogue_query',query)['subjects'] == cat['subjects']
        req = dict(id='cores', predicate='cores', operator='gte', value=20, qualifier='documented')
        assert call('evaluate',dict(query, requirements=[req]))['findings'][0]['status'] == 'supported'
        call('evaluate',dict(query,requirements=[dict(req, predicate='profiles', operator='eq',value='24',qualifier='source_rows_units_unspecified')]),False)
        graph = call('graph_project',dict(source_root=str(root),catalogue=cat))
        assert graph['owner']['engine_version'] == kernel.args.version
        assert call('graph_validate',dict(source_root=str(root),graph=graph))['valid']
        wrong_version = copy.deepcopy(graph);wrong_version['owner']['engine_version']='0.1.0-dev'
        call('graph_validate',dict(source_root=str(root),graph=kernel.seal(wrong_version)),False)
        for mutation in ['date','row']:
            forged = copy.deepcopy(cat)
            if mutation == 'date':
                forged['subjects'][0]['introduced']['through']='2023-01-01'
            else:
                a=next(a for a in forged['subjects'][0]['assertions'] if a['predicate']=='profiles')
                a['value']['rows'][0][2]='2.7'
            forged = kernel.seal(forged)
            call('catalogue_query',dict(query,catalogue=forged),False)
            call('graph_project',dict(source_root=str(root),catalogue=forged),False)
        sub={k:v for k,v in cat['subjects'][0].items() if k!='assertions'}
        profile=kernel.seal(dict(protocol='symphony.shv.coverage-profile.v1',as_of='2026-09-13',selector=dict(op='date',basis='model_introduction',**{'from':'2023-02-01','through':'2026-09-13'})))
        assert call('coverage_plan',dict(profile=profile,subjects=[sub]))['counts']['unresolved']==1
        (root/'s.html').write_text(html.replace('<td>2.0</td>','<td>2.7</td>'))
        call('catalogue_query',query,False)
    print(json.dumps(dict(status='passed',calls=calls,rejections=negative,failures=0,skips=0)))


def test_installed_table_process(prefix):
    # Verifies install-receipt.json and the executable before process tests.
    kernel.args.engine = kernel.installed_engine(prefix)
    test_table_interpretation()


if __name__ == '__main__':
    if kernel.args.prefix:
        test_installed_table_process(kernel.args.prefix)
    else:
        test_table_interpretation()
