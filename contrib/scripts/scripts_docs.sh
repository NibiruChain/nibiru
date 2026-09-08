echo "package.json scripts: (run with yarn)"
cat package.json | jq -r '.scripts'
echo "package.json scriptsComments:"
cat package.json | jq -r '.scriptsComments'