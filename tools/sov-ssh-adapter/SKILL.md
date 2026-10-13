# SOV SSH carrier operation

Read the exact `knowledge/sov/SSH-CARRIER.md` contract before invoking this carrier.
Use qxctl schemas/templates and caller-selected endpoint/runtime/public-key
references. Endpoint validation is local and does not authenticate or connect.
Only explicit `ssh-invoke` starts the selected carrier. Retain its complete output;
`response_lost` requires observation/reconciliation of the same job, never automatic
replay. Target configuration, authority and all scientific inputs remain user owned.

No host creation, listener, private-key extraction, provider call or main merge is
authorized by this document. Qualification records must name actual tested hosts.
