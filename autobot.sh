#!/bin/bash

PID_FILE=".autobot.pid"
LOG_FILE="autobot.log"

# Define the location of your typescript runner
# Assuming you are running this from the project root
CMD="npx ts-node src/index.ts"

function start_bot() {
    if [ -f "$PID_FILE" ]; then
        if ps -p $(cat "$PID_FILE") > /dev/null; then
            echo "Autobot is already running (PID: $(cat $PID_FILE))."
            exit 1
        else
            # Stale PID file
            rm "$PID_FILE"
        fi
    fi

    echo "Starting Git Autobot..."
    
    # Check for API Key
    if [ -z "$AI_API_KEY" ]; then
        echo "Warning: AI_API_KEY environment variable is not set."
        echo "Please export AI_API_KEY='your_key' before running init."
        exit 1
    fi

    # Run in background with nohup
    nohup $CMD > "$LOG_FILE" 2>&1 &
    
    PID=$!
    echo $PID > "$PID_FILE"
    echo "Autobot started with PID $PID."
    echo "Logs are being written to $LOG_FILE"
}

function stop_bot() {
    if [ ! -f "$PID_FILE" ]; then
        echo "Autobot is not running (No PID file found)."
        exit 1
    fi

    PID=$(cat "$PID_FILE")
    if ps -p $PID > /dev/null; then
        kill $PID
        echo "Autobot paused (Process $PID killed)."
    else
        echo "Process $PID not found, cleaning up stale PID file."
    fi
    
    rm "$PID_FILE"
}

case "$1" in
    init)
        start_bot
        ;;
    pause)
        stop_bot
        ;;
    status)
        if [ -f "$PID_FILE" ] && ps -p $(cat "$PID_FILE") > /dev/null; then
            echo "Autobot is RUNNING (PID: $(cat $PID_FILE))"
        else
            echo "Autobot is STOPPED"
        fi
        ;;
    *)
        echo "Usage: ./autobot.sh {init|pause|status}"
        exit 1
        ;;
esac
