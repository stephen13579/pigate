import paho.mqtt.client as mqtt

# Define the callback functions
def on_connect(client, userdata, flags, rc):
    if rc == 0:
        print("Connected successfully")
        client.subscribe("test/topic")
    else:
        print(f"Connection failed with code {rc}")

def on_message(client, userdata, msg):
    print(f"Received message: {msg.payload.decode()} on topic {msg.topic}")

def on_publish(client, userdata, mid):
    print(f"Message {mid} published")

# Create an MQTT client instance
client = mqtt.Client()

# Assign the callback functions
client.on_connect = on_connect
client.on_message = on_message
client.on_publish = on_publish

# Connect to the MQTT broker
client.connect("mqtt-broker", 1883, 60)

# Start the loop
client.loop_start()

# Publish a test message
client.publish("test/topic", "Hello MQTT")

# Keep the client running to listen for messages
try:
    while True:
        pass
except KeyboardInterrupt:
    client.loop_stop()
    client.disconnect()
