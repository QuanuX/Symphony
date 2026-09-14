# AMD product identifier mapping profile v1

Canonical mapping data follows in JSON. This Markdown contract is the reviewed source; no standalone canonical JSON projection or automatic loader is authorized.

```json
{
  "protocol": "symphony.shv.identifier-mapping-profile.v1",
  "profile_id": "amd-product-identifiers",
  "version": "1",
  "issuer": "AMD",
  "issuer_validation": "caller-selected-source-publisher",
  "source_grammar": "shv.normalized-adjacent-fields.v1",
  "heading_section": "div#product-overview",
  "field_section": "article#product-specifications",
  "fields": [
    {
      "predicate": "amd_product_id_boxed",
      "label": "Product ID Boxed",
      "next_label": "Product ID Tray",
      "value_type": "string",
      "qualifier": "issuer=AMD;namespace=product-id-boxed;profile=1"
    },
    {
      "predicate": "amd_product_id_tray",
      "label": "Product ID Tray",
      "next_label": "Supported Technologies",
      "value_type": "string",
      "qualifier": "issuer=AMD;namespace=product-id-tray;profile=1"
    }
  ],
  "comparison": "exact_string_and_qualifier",
  "identity_resolution": "not_performed",
  "digest": "sha256:beabb833ab4bad69969b422a63a25dd79558a59fc8620e3a3e0ac90a13bd8f0e"
}
```
