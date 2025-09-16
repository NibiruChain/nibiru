#!/usr/bin/env bash
set -e

PUBLIC_DIR="../public/docs"

# Check if dist directory exists and is not empty
if [ -d "docs/.vuepress/dist" ] && [ "$(ls -A docs/.vuepress/dist)" ]; then
  echo "Static build (dist) exists and is not empty. Proceeding with copy."

  # Move site contents to monolith public directory
  echo "Moving site contents to home-site public directory."
  rm -rf $PUBLIC_DIR
  mkdir -p $PUBLIC_DIR
  cp -r docs/.vuepress/dist/* $PUBLIC_DIR

  echo "✅ Completed export successfully."
else
  echo "Error: dist directory does not exist or is empty. Aborting."
  exit 1
fi
