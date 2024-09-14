
# File uploader installation

```shell
curl -X POST --user USERNAME:PASSWORD -F file=@upload.html  -v http://localhost:8000/upload
```

Visit http://localhost:8000/upload

# File editor installation

```shell
curl -X POST --user USERNAME:PASSWORD -F file=@editor.html  -v http://localhost:8000/editor

wget https://registry.npmjs.org/monaco-editor/-/monaco-editor-0.51.0.tgz
tar zxvf monaco-editor-0.51.0.tgz
cd package/min/vs/

find . -name '*' -type f -exec curl -u USERNAME:PASSWORD -F "file=@{}" http://localhost:8000/vs/{} \;

cd ../../..
rm monaco-editor-0.51.0.tgz
rm -rf package
```

Visit http://localhost:8000/editor

# Logout javascript snippet

```javascript
location.replace(location.protocol + '//logout:logout@' + location.host)
```