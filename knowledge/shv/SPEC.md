# Symphony Hardware Vector Specification

## Capability Knowledge

Future SHV contracts should be capable of representing, when evidenced:

- exact product, model, revision, stepping, firmware, and release identity;
- CPU architecture, core types and counts, frequencies, instruction facilities, topology, and cache design;
- GPU architecture, memory, compute facilities, and relevant throughput/latency characteristics;
- NIC, fibre, offload, queue, timestamping, and topology capabilities;
- motherboard, chipset, buses, NUMA, memory channels, RAM, and storage design;
- NVMe/controller topology and measurable behavior;
- complete vendor system configurations such as a specifically identified Mac mini model and build;
- benchmarks and measurements with method, environment, sample, time, uncertainty, and provenance;
- known operating-system, compiler, driver, and library compatibility;
- differences among closely named models and configuration variants.

This is an evidence inventory, not a ratified ontology.

## Design Use

SHV may help a user understand how an algorithm used a chipset, GPU, NIC, fibre interface, cache, memory topology, or other physical facility, and what must be reconsidered when porting it to a different Node or TOPS. It exposes possibility and consequence without authoring the algorithm.

## Relationship to SNV and SCV

SNIV identifies a physical Node. SNRV records the resources associated with that Node. SHV supplies versioned capability knowledge about hardware represented by those records. SCV may describe an offsite offering that exposes such hardware. No vector takes another's ownership merely by referencing it.

## Graph and Query Boundary

The planned graph is a rebuildable representation of source and observed evidence. Its C++ engine may validate and query that graph only through ratified operations. AI or private API access must preserve exact version, provenance, compatibility, authorization, and private-installation boundaries.

## Non-Authorization Statement

SHV cannot guarantee latency, infer undisclosed silicon behavior, select hardware for the user, allocate a remote resource, or claim an algorithm is correct because its hardware requirements match.
