#!/bin/sh
INDEX_HTML="/usr/share/nginx/html/index.html"
TMP_INDEX_HTML="/tmp/index.html"
OLD_INDEX_HTML="/tmp/old-index.html"
envsubst < "${INDEX_HTML}" > "${TMP_INDEX_HTML}"
cp "${INDEX_HTML}" "${OLD_INDEX_HTML}"
cp "${TMP_INDEX_HTML}" "${INDEX_HTML}"
