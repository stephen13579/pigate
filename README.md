```
flowchart TB
  subgraph Gate_Controller["Gate Controller"]
    direction TB
        RPI("Raspberry Pi")
        MQTT("MQTT Broker")
        TCP("TCP Connection\nto Remote Server")
        Wiegand("Keypad Input\nWiegand Protocol")
        DB("Local SQLite DB")
  end
  
    MQTT -- Subscribe: Open/Close Commands --> RPI
    RPI -- Publish: Status --> MQTT
    RPI -- Establish --> TCP
    TCP -- Send Updated Passcodes --> RPI
    RPI -- Store Data --> DB
    Wiegand -- Provide Keycode --> RPI
    RPI -- Check Keycode --> DB
    
    classDef mqtt fill:#FFD580,stroke:#333,stroke-width:2px
    classDef tcp fill:#FFB366,stroke:#333,stroke-width:2px
    classDef wiegand fill:#FF8C66,stroke:#333,stroke-width:2px
    classDef db fill:#80B3FF,stroke:#333,stroke-width:2px
    classDef server fill:#6680B3,stroke:#333,stroke-width:2px
    
    MQTT:::mqtt
    TCP:::tcp
    Wiegand:::wiegand
    DB:::db
```