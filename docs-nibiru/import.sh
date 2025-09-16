

LOCAL_DOCS="./docs"
PUBLIC_DOCS="../../docs-nibiru/docs"

copy_docs_dir() {
  local docs_dir=$1 
  local local_docs_dir="$LOCAL_DOCS/$docs_dir"
  rm -rf $local_docs_dir
  mkdir -p $local_docs_dir
  cp -r $PUBLIC_DOCS/$docs_dir $LOCAL_DOCS
}

copy_docs_dir dev
copy_docs_dir ecosystem
copy_docs_dir learn
copy_docs_dir run-nodes
