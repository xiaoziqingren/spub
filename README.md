# spub
a Go repository for sub/pub.

## Usage

```
    // new spub manager
    mc := spub.NewTopics()
	
    // new topic, use spub manager
    if err := mc.NewTopic("happy"); err != nil {
	    // do error control
    }
	
    // subscribe some topic, use spub manager
    myChan := make(chan int)
	
    sub, err := mc.Subscribe("happy", myChan)
    if err != nil {
	    // do error control
    }
	
    // unsubscribe some topic, use sub which get from subscribe
    sub.Unsubscribe()
	
    // publish event to the topic, use spub manager
    if err := mc.Publish("happy", 0x100); err != nil {
	    // do error control
    }
```
