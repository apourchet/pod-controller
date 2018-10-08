import requests
import time
import utils


def test_healthy_forever():
    utils.start_controller('healthy_forever.json')

    for i in range(3):
        resp = requests.get('http://localhost:8888/healthy')
        assert resp is not None and resp.json() is not None
        data = resp.json()
        assert data['healthy']
        time.sleep(1)

    utils.kill_controller()


def test_healthy_double_forever():
    utils.start_controller('healthy_double_forever.json')

    for i in range(3):
        resp = requests.get('http://localhost:8888/healthy')
        assert resp is not None and resp.json() is not None
        data = resp.json()
        assert data['healthy']
        time.sleep(1)

    utils.kill_controller()


def test_unhealthy_forever():
    utils.start_controller('unhealthy_forever.json')

    # Wait for it to get into its steady state.
    for i in range(5):
        resp = requests.get('http://localhost:8888/status')
        assert resp is not None and resp.json() is not None
        data = resp.json()
        if len(data[0]['States']) == 3:
            break
        time.sleep(1)

    # Then query the healthy bit.
    for i in range(3):
        resp = requests.get('http://localhost:8888/healthy')
        assert resp is not None and resp.json() is not None
        data = resp.json()
        assert not data['healthy']
        time.sleep(1)

    utils.kill_controller()
