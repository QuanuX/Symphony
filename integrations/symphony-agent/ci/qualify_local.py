#!/usr/bin/env python3
"""Build/test one pre-provisioned native platform; never install dependencies or publish.

Run on the target itself. A successful container run is container evidence only.
No Symphony service or provider is contacted by this script.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import shutil
import subprocess
import struct
import sys
from datetime import datetime, timezone


def version_tuple(text):
    found = re.search(r'(\d+)\.(\d+)(?:\.(\d+))?', text)
    if not found:
        raise ValueError('No semantic version found')
    return tuple(int(x or 0) for x in found.groups())


def system_facts():
    facts = {'system': platform.system(), 'machine': platform.machine(), 'kernel': platform.release(),
             'python': platform.python_version(), 'uid': os.getuid(), 'euid': os.geteuid()}
    if Path('/etc/os-release').is_file():
        facts['os_release'] = dict(line.split('=',1) for line in Path('/etc/os-release').read_text().splitlines() if '=' in line and not line.startswith('#'))
        facts['os_release'] = {k:v.strip('"') for k,v in facts['os_release'].items()}
    facts['wsl_evidence'] = {'kernel_contains_microsoft': 'microsoft' in platform.release().lower(), 'distribution': os.environ.get('WSL_DISTRO_NAME')}
    return facts


def elf_architecture(path):
    with path.open('rb') as stream:
        header = stream.read(20)
    if len(header) != 20 or header[:4] != b'\x7fELF' or header[4] != 2 or header[5] not in (1, 2):
        raise RuntimeError('Expected a 64-bit ELF executable')
    machine = struct.unpack('<H' if header[5] == 1 else '>H', header[18:20])[0]
    if machine not in (62, 183):
        raise RuntimeError('Unsupported ELF machine: '+str(machine))
    return {62:'x86_64',183:'arm64'}[machine]


def source_inventory(source):
    # Include build rules, test expectations and adapter scripts as well as C++.
    files = [source/name for name in ('CMakeLists.txt', 'README.md', 'LICENSE', 'plugin.json', 'mcp.json', '.mcp.json') if (source/name).is_file()]
    for folder in ('src', 'include', 'third_party', 'scripts', 'tests', 'skills', 'docs', 'adapters', '.claude-plugin'):
        directory=source/folder
        if directory.is_symlink():raise RuntimeError('Source directory must not be a symlink')
        for path in directory.rglob('*'):
            if path.is_symlink():raise RuntimeError('Source inventory must not omit symlinked entries')
            if path.is_file() and '__pycache__' not in path.parts and path.suffix != '.pyc':files.append(path)
    if any(path.is_symlink() for path in files):
        raise RuntimeError('Source inventory must contain regular files, not symlinks')
    return {str(path.relative_to(source)):hashlib.sha256(path.read_bytes()).hexdigest() for path in sorted(files)}


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--source',required=True,type=Path)
    p.add_argument('--out',required=True,type=Path,help='A new directory for this run')
    p.add_argument('--compiler',required=True,help='Exact already-installed C++ compiler path')
    p.add_argument('--cmake',default='cmake')
    p.add_argument('--ctest',default='ctest')
    p.add_argument('--expected-os',required=True,choices=['ubuntu','macos'])
    p.add_argument('--expected-arch',required=True,choices=['x86_64','arm64'])
    p.add_argument('--ubuntu-version',choices=['22.04','24.04','26.04'])
    p.add_argument('--host-kind',required=True,choices=['container','vm','physical','wsl2'])
    p.add_argument('--sanitizers',action='store_true')
    p.add_argument('--macos-deployment-target',help='Explicit minimum macOS version for Apple builds')
    p.add_argument('--jobs',type=int,default=2)
    args=p.parse_args()
    if sys.version_info < (3,11):p.error('Use Python3.11+; do not silently use Ubuntu22.04 stock Python3.10')
    if not 1<=args.jobs<=16:p.error('Use 1..16 jobs')
    source=args.source.resolve(strict=True)
    if not (source/'CMakeLists.txt').is_file():p.error('Source package is missing CMakeLists.txt')
    # Preserve clang++/g++ argv[0]; following its symlink can select the C driver.
    compiler=Path(os.path.abspath(args.compiler))
    if not compiler.is_file() or not os.access(compiler,os.X_OK):p.error('Select an executable compiler')
    cmake=shutil.which(args.cmake);ctest=shutil.which(args.ctest)
    if not cmake or not ctest:p.error('Select installed CMake and CTest')
    make=shutil.which('make')
    if not make:p.error('Install Make before qualification; the helper uses the Unix Makefiles generator')
    if args.expected_os=='macos' and (not args.macos_deployment_target or not re.fullmatch(r'\d+\.\d+(?:\.\d+)?',args.macos_deployment_target)):
        p.error('Select an explicit --macos-deployment-target such as 14.0; qualify that minimum separately')
    if args.out.resolve().is_relative_to(source):p.error('Keep output outside the source tree')
    args.out.mkdir(parents=True,exist_ok=False)
    out=args.out.resolve()
    report={'schema':'symphony.platform-qualification.v1','started_at':datetime.now(timezone.utc).isoformat(),
            'host_kind_operator_declared':args.host_kind,'facts':system_facts(),'source':str(source),
            'scope':'bridge_native_fixture_tests_only','owner_installation_qualified':False,
            'agent_host_qualified':False,'wsl2_windows_host_qualified':False,
            'native_hardware_identity_independently_verified':False,'status':'running','commands':[]}
    report['compiler_invocation_path']=str(compiler)
    report['compiler_resolved_path']=str(compiler.resolve(strict=True))
    def save(): (out/'RESULTS.json').write_text(json.dumps(report,indent=2)+'\n')
    def run(label,argv,timeout=300):
        report['commands'].append({'label':label,'argv':list(map(str,argv))});save()
        with (out/(label+'.log')).open('wb') as log:
            result=subprocess.run(list(map(str,argv)),cwd=source,stdout=log,stderr=subprocess.STDOUT,stdin=subprocess.DEVNULL,timeout=timeout,check=False)
        report['commands'][-1]['exit_code']=result.returncode;save()
        if result.returncode:raise RuntimeError(label+' failed; inspect its log')
        return (out/(label+'.log')).read_text(errors='replace')
    try:
        facts=report['facts'];machine={'aarch64':'arm64','arm64':'arm64','amd64':'x86_64','x86_64':'x86_64'}.get(facts['machine'].lower())
        if machine!=args.expected_arch:raise RuntimeError('Wrong executing architecture; cross-build is not a native qualification')
        if facts['euid']==0:raise RuntimeError('Run qualification as an unprivileged user; provision the container separately')
        if args.expected_os=='ubuntu':
            distro=facts.get('os_release',{})
            if facts['system']!='Linux' or distro.get('ID')!='ubuntu' or not args.ubuntu_version or distro.get('VERSION_ID')!=args.ubuntu_version:raise RuntimeError('Actual Ubuntu release does not match the selected target')
        elif facts['system']!='Darwin':raise RuntimeError('Apple qualification requires a macOS runner')
        if args.host_kind=='wsl2' and not facts['wsl_evidence']['kernel_contains_microsoft']:raise RuntimeError('This is not observed WSL; a generic Linux container cannot qualify WSL2')
        if args.host_kind=='wsl2':report['wsl_scope']='Microsoft kernel observed only; WSL generation and container nesting require separate Windows-side evidence'
        if args.expected_os=='macos':
            report['apple_sdk']=run('apple-sdk',['/usr/bin/xcrun','--sdk','macosx','--show-sdk-version']).strip()
            translated=subprocess.run(['/usr/sbin/sysctl','-in','sysctl.proc_translated'],capture_output=True,text=True)
            report['rosetta_translated']=translated.stdout.strip()=='1'
            if report['rosetta_translated']:raise RuntimeError('Native Apple qualification cannot run through Rosetta')
        cmake_text=run('cmake-version',[cmake,'--version']);ctest_text=run('ctest-version',[ctest,'--version'])
        if version_tuple(cmake_text)<(3,30,0) or version_tuple(ctest_text)!=version_tuple(cmake_text):raise RuntimeError('CMake>=3.30 and matching CTest are required')
        report['compiler_version']=run('compiler-version',[compiler,'--version']).strip()
        report['make_version']=run('make-version',[make,'--version']).strip()
        tree=source_inventory(source)
        (out/'source-sha256.json').write_text(json.dumps(tree,indent=2)+'\n')
        build=out/'build'
        configure=[cmake,'-S',source,'-B',build,'-G','Unix Makefiles','-DCMAKE_MAKE_PROGRAM='+make,'-DCMAKE_BUILD_TYPE='+('Debug' if args.sanitizers else 'Release'),'-DCMAKE_CXX_COMPILER='+str(compiler),'-DPython3_EXECUTABLE='+sys.executable,'-DSYMPHONY_SANITIZERS='+('ON' if args.sanitizers else 'OFF')]
        if facts['system']=='Darwin':
            report['macos_deployment_target']=args.macos_deployment_target
            configure += ['-DCMAKE_OSX_ARCHITECTURES='+args.expected_arch,'-DCMAKE_OSX_DEPLOYMENT_TARGET='+args.macos_deployment_target]
        run('configure',configure)
        run('build',[cmake,'--build',build,'--parallel',str(args.jobs)],timeout=900)
        run('ctest',[ctest,'--test-dir',build,'--output-on-failure'],timeout=600)
        details=(build/'Testing/Temporary/LastTest.log').read_text(errors='replace')
        (out/'test-details.log').write_text(details)
        if re.search(r'\bskipped\b|\bNOT_RUN\b|\bNot Run\b',details,re.I):raise RuntimeError('Tests skipped: resolve the missing boundary; do not call this target qualified')
        for marker in ['transport checks passed','hash checks passed','Ran 20 tests','Ran 4 tests']:
            if marker not in details:raise RuntimeError('Expected focused suite missing: '+marker)
        binary=build/'symphony-mcp';report['binary_sha256']=hashlib.sha256(binary.read_bytes()).hexdigest()
        if facts['system']=='Linux':report['binary_architecture']=elf_architecture(binary)
        else:report['binary_architecture']=run('macho-architecture',['/usr/bin/lipo','-archs',binary]).strip()
        if report['binary_architecture']!=args.expected_arch:raise RuntimeError('Built binary architecture differs from selected native target')
        run('binary-version',[binary,'--version'])
        if facts['system']=='Linux':
            run('elf-header',['readelf','-h',binary]);run('elf-dynamic',['readelf','-d',binary]);run('elf-symbols',['readelf','--version-info',binary])
            run('elf-load-segments',['readelf','-lW',binary])
            deps=run('runtime-linkage',['ldd',binary])
            if 'not found' in deps:raise RuntimeError('Unresolved runtime library')
        else:
            run('runtime-linkage',['otool','-L',binary])
            run('macho-load-commands',['otool','-l',binary])
        report['status']='bridge_tests_passed_on_observed_host'
        report['note']='No cross-OS, GPU, WSL Windows-host, native-owner authority, or agent-host certification is implied.'
        save();print(json.dumps({'results':str(out/'RESULTS.json'),'status':report['status']},indent=2))
    except Exception as error:
        report['status']='failed';report['error']=str(error);save();raise

if __name__=='__main__':
    main()
