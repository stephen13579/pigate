import unittest
from unittest.mock import Mock, patch
from context import MQTTClient

class TestMQTTClient(unittest.TestCase):
    def setUp(self):
        self.broker_address = "test_broker"
        self.username = "user"
        self.password = "pass"
        self.client = MQTTClient(self.broker_address, username=self.username, password=self.password)

    @patch('paho.mqtt.client.Client.connect')
    def test_connect(self, mock_connect):
        self.client.connect()
        mock_connect.assert_called_once_with(self.broker_address, 1883, 60)

    @patch('paho.mqtt.client.Client.subscribe')
    def test_subscribe(self, mock_subscribe):
        self.client.subscribe("test/topic", qos=1)
        mock_subscribe.assert_called_once_with("test/topic", 1)

    @patch('paho.mqtt.client.Client.publish')
    def test_publish(self, mock_publish):
        self.client.publish("test/topic", "message", qos=1, retain=True)
        mock_publish.assert_called_once_with("test/topic", "message", 1, True)

if __name__ == '__main__':
    unittest.main()
