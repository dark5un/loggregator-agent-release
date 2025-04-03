#!/bin/bash

# convert-mocks.sh
# A helper script for migrating from hel to mockery mocking library
# 
# Usage:
#   ./scripts/convert-mocks.sh [directory]
#
# If no directory is specified, the script will search the entire src directory.

set -e

TARGET_DIR=${1:-"src"}
echo "Scanning for hel directives in ${TARGET_DIR}..."

# Find all //go:generate hel directives
HEL_FILES=$(grep -r "//go:generate hel" --include="*.go" ${TARGET_DIR} | cut -d: -f1 | sort | uniq)

if [ -z "$HEL_FILES" ]; then
  echo "No files with hel directives found."
  exit 0
fi

echo "Found $(echo "$HEL_FILES" | wc -l | tr -d ' ') files with hel directives."
echo

# Process each file with hel directives
for FILE in $HEL_FILES; do
  echo "Processing $FILE..."
  
  # Extract package name
  PACKAGE=$(grep -m 1 "package" $FILE | cut -d ' ' -f 2)
  
  # Replace hel directives with mockery directives
  sed -i.bak 's|//go:generate hel|//go:generate mockery --name|g' $FILE
  
  # Check if there are helheim_test.go files in the directory
  DIR=$(dirname $FILE)
  HELHEIM_FILES=$(find $DIR -name "helheim_test.go")
  
  if [ -n "$HELHEIM_FILES" ]; then
    echo "  Found helheim_test.go files in $DIR. Removing them..."
    for HELHEIM in $HELHEIM_FILES; do
      echo "  Removing $HELHEIM..."
      rm $HELHEIM
    done
  fi
  
  # Run go generate to create mockery mocks
  echo "  Running go generate for $FILE..."
  (cd $DIR && go generate $FILE)
  
  # Clean up backup files
  rm -f $FILE.bak
done

echo
echo "Migration complete. Please check the following items manually:"
echo "1. Update tests to use the mockery pattern (method expectations instead of channels)"
echo "2. Ensure thread safety in mocks where needed"
echo "3. Add a MockTesting helper if using Ginkgo"
echo "4. Run tests with race detection to identify any race conditions"
echo 