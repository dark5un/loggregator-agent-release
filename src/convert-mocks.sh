#!/bin/bash
set -e

# Generate mocks using mockery
go generate ./...

# Create a directory to save old test files for reference
mkdir -p .old_test_files

# Update test files to use new mock patterns
find ./pkg -name "*_test.go" | xargs grep -l "newMock" | while read file; do
  # Save old file for reference
  cp "$file" ".old_test_files/$(basename $file)"
  
  # Replace mockery mocks - general pattern
  # mockery generates NewMock{Interface} instead of newMock{Interface}
  sed -i '' 's/newMock\([A-Z][a-zA-Z]*\)/New\1Mock/g' "$file"
  
  # We also need to update how the mocks are used - hel uses channels, mockery uses methods
  echo "Updating $file..."
done

echo "Conversion complete. Check test files and run tests to verify." 