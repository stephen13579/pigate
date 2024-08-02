import paho.mqtt.client as mqtt

class MQTTClientInterface:
    def connect(self): pass
    def disconnect(self): pass
    def subscribe(self, topic: str, qos: int = 0): pass
    def publish(self, topic: str, payload: str, qos: int = 0, retain: bool = False): pass

class MQTTClient(MQTTClientInterface):
    def __init__(self, broker_address: str, port: int = 1883, username: str = None, password: str = None):
        self.client = mqtt.Client()
        self.broker_address = broker_address
        self.port = port
        if username and password:
            self.client.username_pw_set(username, password)
        self.client.on_connect = self.on_connect
        self.client.on_message = self.on_message

    def connect(self):
        self.client.connect(self.broker_address, self.port, 60)
        self.client.loop_start()

    def disconnect(self):
        self.client.loop_stop()
        self.client.disconnect()

    def subscribe(self, topic: str, qos: int = 0):
        self.client.subscribe(topic, qos)

    def publish(self, topic: str, payload: str, qos: int = 0, retain: bool = False):
        self.client.publish(topic, payload, qos, retain)

    def on_connect(self, client, userdata, flags, rc):
        print(f"Connected with result code {rc}")

    def on_message(self, client, userdata, msg):
        print(f"Received message: {msg.payload.decode()} on topic {msg.topic}")
        self.process_message(msg.topic, msg.payload.decode())

    def process_message(self, topic: str, message: str):
        pass  # Implement message processing logic here
