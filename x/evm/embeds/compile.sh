#!/usr/bin/env bash

# 1 | Get all .sol files in the current directory
SOL_FILES=$(find . -maxdepth 1 -name "*.sol")

remove_suffix_from() {
    local string="$1"
    local suffix="$2"
    echo "${string%"$suffix"}"
}

compile_contract() {
  local contract_name="$1"
  echo "Compiling contract \"$contract_name\""

  cp "$contract_name" "contracts/$contract_name"
  npx hardhat compile

  contract_name_no_file_ext=$(remove_suffix_from "$contract_name" ".sol")
  local artifact_fname="artifacts/contracts/$contract_name/${contract_name_no_file_ext}.json"
  cp "$artifact_fname" "${contract_name_no_file_ext}Compiled.json"
}

npm i --check-files 2>/dev/null 1>/dev/null
rm -f contracts/Lock.sol

# 2 | Iterate through each file
for precompile_interface in $SOL_FILES
do
    # Store the file name in a variable
    filename=$(basename "$precompile_interface")
    
    # Call the "foo" function with the filename
    # Replace this line with the actual "foo" function or command you want to run
    compile_contract "$filename"
    
    # Optional: Print the filename being processed
    echo "Processed: $filename"
done

# 3 | Echo a success message
echo "Successfully processed all .sol files in the current directory!"
