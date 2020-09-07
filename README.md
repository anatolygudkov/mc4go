# mCounters for golang
Counters for midi- and microservices.

## Motivation and goals
It's a common problem to collect and expose metrics from your midi- and microservice applications. This library tryes to help you whith it.

The library is aimed to be:
 - as fast as possible. It uses memory mapped files and direct memory access to store the counters.
 - service implementation agnostic. For example, you can expose JMX metrics from your Java application and read them from a Golang sidecar application or vise versa (see https://github.com/anatolygudkov/mc4j).
 - usable to expose static information about the application as well as dynamic counters.
 - a 0-dependency project. Just copy-paste the sources into your project.

## Usage
*The library isn't tested for Windows yet*
### How to write counters
```
//Here is some statics we'd like to expose
statics := make(map[string]string)
statics["my.prop.1"] = "value 1"
statics["my.prop.2"] = "value 2"

w, err := NewWriterForName("mycounters.dat", statics, 100)
if err != nil {
	return err
}
defer w.Close()

//First counter
myCounter1, err := w.AddCounter("my.cnt.1")
if err != nil {
	return err
}
myCounter1.Set(100)
myCounter1.Close() //Free the memory allocated for the counter

//Second counter
myCounter2, err := w.AddCounter("my.cnt.2")
if err != nil {
	return err
}
myCounter2.Increment()
 
```

### How to read counters
```
r, err := mc4go.NewReaderForName("mycounters.dat")
if err != nil {
	return err
}
defer r.Close()

fmt.Printf("version: %d\n", r.Version())
fmt.Printf("pid: %d\n", r.Pid())
fmt.Printf("started: %d\n", r.StartTime())

r.ForEachStatic(func(label, value string) bool {
	fmt.Printf("static: %s=%s\n", label, value)
	return true
})

r.ForEachCounter(func(id, value int64, label string) bool {
	fmt.Printf("counter: %s[%d]=%d\n", label, id, value)
	return true
}) 
```

There are two reades included into the library:
- https://github.com/anatolygudkov/mc4go/blob/master/cmd/mcprinter/mcprinter.go - prints out the content of a counters' file to console
- https://github.com/anatolygudkov/mc4go/blob/master/cmd/mcendpoint/mcendpoint.go - exposes the content of a counters' file with a simple RESTful API

## Concurrency issues
- Counters are thread safe and one counter can be modified in different goroutines.
- After a counter is closed, it must be not used, since its memory slot can be occupied by a new counter and the value of that new counter will be modified unexpectedtly.
- Counters must not be modified after the writer is closed, because such modification leads to a segmentation fault.

## License
The code is available under the terms of the [MIT License](http://opensource.org/licenses/MIT).
