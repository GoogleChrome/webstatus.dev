Regenerating tar ball archives

```sh
testcases=( "case01_basic" "case02_nested" "case03_empty" )
pushd uncompressed
for testcase in "${testcases[@]}"
do
    echo "$testcase"
    tar -czvf ../compressed/$testcase.tar.gz $testcase
done
popd
```
