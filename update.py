import os
import time
import requests
import subprocess
import signal
from dotenv import dotenv_values, set_key

ENV_FILE = ".env"
VERSION_KEY = "APP_VERSION"
PID_FILE = "server.pid"
LOG_FILE = "updater.log"
LOCAL_VER_FILE = "version.txt"
SERVER_BINARY = "./nas-server"
BUILD_TARGET = "nas-server"
REMOTE_VER_URL = "https://raw.githubusercontent.com/raunaksingh9800/go_update_sys/main/version.txt"
UPDATE_INTERVAL = 60*10 # seconds



def log(msg):
    entry = f"[{time.strftime('%Y-%m-%dT%H:%M:%S')}] {msg}\n"
    print(entry, end="")
    with open(LOG_FILE, "a") as f:
        f.write(entry)


def read_local_version():
    config = dotenv_values(ENV_FILE)
    return config.get(VERSION_KEY, "")


def fetch_remote_version():
    try:
        resp = requests.get(REMOTE_VER_URL, timeout=10)
        if resp.status_code == 200:
            return resp.text.strip()
        log(f"Failed to fetch remote version: HTTP {resp.status_code}")
    except Exception as e:
        log(f"Error fetching remote version: {e}")
    return ""


def update_env_version(new_version):
    set_key(ENV_FILE, VERSION_KEY, new_version)


def update_local_version_txt(new_version):
    with open(LOCAL_VER_FILE, "w") as f:
        f.write(new_version)


def run_command(*args):
    try:
        result = subprocess.run(args, check=True, capture_output=False)
        return result.returncode == 0
    except subprocess.CalledProcessError as e:
        log(f"Command {' '.join(args)} failed: {e}")
        return False


def stop_server():
    if not os.path.exists(PID_FILE):
        log("No PID file found. Server may not be running.")
        return
    try:
        with open(PID_FILE, "r") as f:
            pid = int(f.read().strip())
        os.kill(pid, signal.SIGTERM)
        log(f"Successfully killed server process with PID: {pid}")
    except Exception as e:
        log(f"Failed to stop server: {e}")
    finally:
        if os.path.exists(PID_FILE):
            os.remove(PID_FILE)


def build_server():
    log("Building NAS server...")
    return run_command("go", "build", "-o", BUILD_TARGET, "main.go")


def start_server():
    log("Starting NAS server...")
    try:
        proc = subprocess.Popen([SERVER_BINARY])
        with open(PID_FILE, "w") as f:
            f.write(str(proc.pid))
        return True
    except Exception as e:
        log(f"Failed to start server: {e}")
        return False


def is_server_running():
    if not os.path.exists(PID_FILE):
        return False
    try:
        with open(PID_FILE, "r") as f:
            pid = int(f.read().strip())
        os.kill(pid, 0)  # Check if process exists
        return True
    except Exception:
        return False

def run_command(*args):
    try:
        result = subprocess.run(args, check=True, capture_output=False)
        return result.returncode == 0
    except subprocess.CalledProcessError as e:
        log(f"Command {' '.join(args)} failed: {e}")
        return False

# ... [unchanged functions above]

def main():
    while True:
        log("Checking for updates...")

        local_ver = read_local_version()
        remote_ver = fetch_remote_version()

        if remote_ver and remote_ver != local_ver:
            log(f"Update found: {local_ver} → {remote_ver}")
            stop_server()

            if not run_command("git", "fetch", "origin"):
                goto_sleep()
                continue
            if not run_command("git", "reset", "--hard", "origin/main"):
                goto_sleep()
                continue

            # 🆕 Step: Update go.mod and go.sum
            log("Tidying Go modules...")
            if not run_command("go", "mod", "tidy"):
                log("Failed to tidy Go modules. Aborting update.")
                goto_sleep()
                continue

            if not build_server():
                goto_sleep()
                continue

            update_env_version(remote_ver)
            update_local_version_txt(remote_ver)

            if start_server():
                log("Server updated and running!")
            else:
                log("Failed to start server after update.")
        else:
            log("No update needed.")
            if not is_server_running():
                log("Server is not running. Starting it now...")
                if start_server():
                    log("Server started successfully.")
                else:
                    log("Failed to start server.")

        goto_sleep()

def goto_sleep():
    log(f"Sleeping for {UPDATE_INTERVAL} seconds...\n")
    time.sleep(UPDATE_INTERVAL)


if __name__ == "__main__":
    main()



