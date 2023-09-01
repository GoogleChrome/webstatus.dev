
```
TEMP_DIR=$(mktemp -d)
wget -q -O ${TEMP_DIR}/go-jsonschema_Linux_x86_64.tar.gz \
    https://github.com/omissis/go-jsonschema/releases/download/v0.12.1/go-jsonschema_Linux_x86_64.tar.gz
tar -xf ${TEMP_DIR}/go-jsonschema_Linux_x86_64.tar.gz -C ${TEMP_DIR}
mv ${TEMP_DIR}/go-jsonschema /usr/local/go/bin/
rm -rf $TEMP_DIR
```

```
wget https://raw.githubusercontent.com/web-platform-dx/feature-set/main/schemas/defs.schema.json
go-jsonschema defs.schema.json -p schemas -o schemas/feature_set.go -e
```

```sh
oapi-codegen -config server.cfg.yaml openapi.yaml
oapi-codegen -config types.cfg.yaml openapi.yaml
```


```
quicktype \
  --src schemas/defs.schema.json \
  --src-lang schema \
  --lang go \
  --top-level FeatureData \
  --out schemas/feature_data.go \
  --package schemas \
  --field-tags json
```