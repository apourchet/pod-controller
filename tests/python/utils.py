import requests
import subprocess
import time


def start_controller(spec, runtime=None, port=None):
    kill_controller(port or 8888)

    command = ['./bins/controller', '--spec', 'tests/specs/' + spec]
    if runtime is not None:
        command.extend(['--runtime', runtime])
    if port is not None:
        command.extend(['--port', port])
    subprocess.Popen(command)

    if port is None:
        port = 8888

    for i in range(100):
        try:
            requests.get('http://localhost:{}/'.format(port))
            return
        except Exception:
            time.sleep(0.1)


def kill_controller(port=8888):
    for i in range(100):
        try:
            requests.get('http://localhost:{}/kill'.format(port))
            time.sleep(0.1)
        except Exception:
            return
