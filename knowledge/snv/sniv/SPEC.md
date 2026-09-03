# Symphony Node Identity Vector Specification

## Identity Rules

- A destroyed or physically replaced resource is a new Node.
- A separately established provider resource with a different provider-resource identifier is a new Node even when its configuration is nearly identical to a retired resource.
- A materially changed local hardware resource receives a new physical-resource identity even if a provider identifier does not change.
- A software-only reflash, operating-system replacement, restart, disconnection, or reconnection does not alone create a new physical Node.
- A remotely attached or virtual resource does not alone change the local physical Node identity.
- A Node incarnation records establishment of the Node on a bus in a trading, research, or other Symphony system; it does not replace physical identity.

A provider's establishment of a separate provider-resource identifier creates the new Node boundary stated above even though the identifier alone does not prove the resource's undisclosed physical substrate. Provider display labels, serials, firmware evidence, Symphony identifiers, offering references, infrastructure-domain associations, and user assertions otherwise remain distinguishable evidence. Their exact precedence, conflict, and required-field rules remain deferred. No nickname or provider display label is automatically conclusive.

An alleged resource without an applicable identity remains unresolved or theoretical. SNIV must not manufacture identity in order to make another workflow proceed.

## Non-Authorization Statement

SNIV establishes no provider ownership, legal ownership, operational permission, cluster membership, resource compatibility, or name allocation.
