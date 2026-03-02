#!/bin/bash

# Default values
DIRECTORY="."
INTERVAL=5

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -Directory)
            DIRECTORY="$2"
            shift 2
            ;;
        -Interval)
            INTERVAL="$2"
            shift 2
            ;;
        *)
            shift
            ;;
    esac
done

cd "$(dirname "$0")"
cd ../..

echo -e "\033[32mStarting continuous test runner...\033[0m"
echo "Target directory: $DIRECTORY"
echo "Interval: $INTERVAL seconds"
echo "Press Ctrl+C to stop."
echo ""

while true; do
    TIMESTAMP=$(date +"%H:%M:%S")
    
    if [ -d "$DIRECTORY" ]; then
        pushd "$DIRECTORY" > /dev/null
        
        echo -e "\033[36m[$TIMESTAMP] Cleaning test cache...\033[0m"
        go clean -testcache
        
        echo -e "\033[35m[$TIMESTAMP] Running tests...\033[0m"
        go test ./...
        
        popd > /dev/null
    else
        echo -e "\033[31mError: Directory '$DIRECTORY' not found.\033[0m"
        break
    fi

    echo -e "\n\033[2m--- Waiting $INTERVAL seconds ---\033[0m"
    sleep $INTERVAL
done
