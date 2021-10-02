# pipe <a href="https://johnstarich.com/go/pipe"><img src="https://img.shields.io/badge/gopages-reference-%235272B4" /></a>

Pipes handle errors so you don't have to.

This means:
* Verifying program correctness is easier
* Fewer `if` branches makes the program simpler

Want to get going? Jump to the [quick start](#quick-start).

For more detail on how this works, read through [the approach](#more-rigorous-tests-with-less-effort) below.
For full documentation, check out [the reference](https://johnstarich.com/go/pipe).

**Caveat:** `Pipe`s should probably not be used _everywhere_. Instead, they should be used when stitching together many different failable functions. They're especially relevant when those functions are in a separate library.

## Quick start

A `Pipe` is a series of functions that pass data to one another, in-order. These functions can also fail with an error, which stops the flow and returns immediately.

In this quick start, we'll create a pipe that returns true if the string argument parses to a positive integer.

1. First, create a `Pipe` named `isPositive`. You can pass additional options for this pipe, but we'll use the defaults for now.
```go
import "github.com/johnstarich/go/pipe"

isPositive := pipe.New(pipe.Options{}) // using default options
```

2. Append a function to convert from `args []interface{}` into the `string` we're expecting.
```go
isPositive = isPositive.Append(func(args []interface{}) string {
    return args[0].(string) // for this example, just coerce the type
})
```

Every time we append a function, the return types and parameter types must match. Then `Append()` returns a new Pipe with the added operation.

3. Append a function to parse the previous string into an integer. The return types from the previous step become the new input types.
```go
isPositive = isPositive.Append(func(s string) (int64, error) {
    return strconv.ParseInt(s, 10, 64)
})
```

4. Finally, append a function to check if the integer is positive. Notice we didn't need to handle the error from the previous step.
```go
isPositive = isPositive.Append(func(i int64) bool {
    return i > 0
})
```

5. The pipe is ready to use. Let's run it.
```go
out, err := isPositive.Do("42")
if err != nil {
    panic(err)
}
positive := out[0].(bool)
fmt.Println("42 is positive =", positive)
// Output: 42 is positive = true
```

We covered a lot of ground there. Let's take a look at the full example:
```go
import "github.com/johnstarich/go/pipe"

isPositive := pipe.New(pipe.Options{}).
    Append(func(args []interface{}) string {
	return args[0].(string) // convert to expected input types
    }).
    Append(func(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64) // parse input string to an integer
    }).
    Append(func(i int64) bool {
	return i > 0 // check if it's positive
    })

out, err := isPositive.Do("42") // run with input "42"
if err != nil {
    panic(err)
}
positive := out[0].(bool)
fmt.Println("42 is positive =", positive)
// Output: 42 is positive = true
```

Our new pipeline is successfully checking a string is a positive integer!

You may have noticed we didn't use an `if` statement to handle the `error` from `strconv.ParseInt()` -- that's important. By using a pipeline, that error handling is already taken care of.

We'll go over why this is significant in the next section.

## More rigorous tests with less effort

`Pipe`s are heavily tested with known outcomes when functions return errors. Since internal `Pipe` behavior is fully vetted, it isn't necessary for us to write rigorous error path tests to achieve similar levels of correctness.

By eliminating `if` branches in our own code, we're also eliminating branches we must test ourselves. The branches still exist, sure, but they only reside in `Pipe`'s built-in error handling, _which already has tests_.

`Pipe`s help clean up code interacting with error-prone systems, like networks or files. In the below example, we reach out to `api.github.com`, read the body, parse it, and return the account creation date as a `time.Time`.

```go
var userCreatedDatePipe = pipe.New(pipe.Options{}).
    // create the request
    Append(func(args []interface{}) (*http.Request, error) {
	ctx := args[0].(context.Context)
	githubName := args[1].(string)
        return http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/users/"+githubName, nil)
    }).
    // run the request
    Append(http.DefaultClient.Do).
    // verify success, return body reader
    Append(func(resp *http.Response) (io.Reader, error) {
	return resp.Body, pipe.CheckErrorf(resp.StatusCode != http.StatusOK, "fetch from api.github.com failed with status: %s", resp.Status)
    }).
    // read body
    Append(ioutil.ReadAll).
    // parse out created date
    Append(func(body []byte) (string, error) {
	var model struct {
	    CreatedAt string `json:"created_at"`
	}
	err := json.Unmarshal(body, &model)
	return model.CreatedAt, err
    }).
    // parse date into time.Time
    Append(func(rawDate string) (time.Time, error) {
	return time.Parse(time.RFC3339, rawDate)
    })

func UserCreatedDate(ctx context.Context, githubName string) (time.Time, error) {
    out, err := userCreatedDatePipe.Do(ctx, githubName)
    var createdDate time.Time
    if err == nil {
	createdDate = out[0].(time.Time)
    }
    return createdDate, err
}

name := "octocat"
createdDate, err := UserCreatedDate(context.Background(), name)
if err != nil {
    panic(err)
}
fmt.Println("Account", name, "created on", createdDate.Format(time.Stamp))
// Output:
// Account octocat created on Jan 25 18:44:36
```

Pretty slick, eh?

For comparison, see the traditional error handling approach below. Notice there are now several additional branches handling errors.

Some of these errors can be difficult to hit in a test, making it difficult to verify correctness on every possible branch.

```go
func UserCreatedDate(ctx context.Context, githubName string) (time.Time, error) {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/users/"+githubName, nil)
    if err != nil {
	return time.Time{}, err
    }
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
	return time.Time{}, err
    }
    if resp.StatusCode != http.StatusOK {
	return time.Time{}, fmt.Errorf("fetch from api.github.com failed with status: %s", resp.Status)
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
	return time.Time{}, err
    }

    var model struct {
	CreatedAt string `json:"created_at"`
    }
    err = json.Unmarshal(body, &model)
    if err != nil {
	return time.Time{}, err
    }
    return time.Parse(time.RFC3339, rawDate)
}

name := "octocat"
createdDate, err := UserCreatedDate(context.Background(), name)
if err != nil {
    panic(err)
}
fmt.Println("Account", name, "created on", createdDate.Format(time.Stamp))
```

The difference may not be immediately obvious. Let's make it clearer by analyzing branch coverage of both programs.

Assume we wrote 1 test for the success path using both Pipes and traditional error handling, then collected their test coverage results.

Using Pipes with 1 success path test:
```diff
 var userCreatedDatePipe = pipe.New(pipe.Options{}).
     Append(func(args []interface{}) (*http.Request, error) {
+	ctx := args[0].(context.Context)
+	githubName := args[1].(string)
+       return http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/users/"+githubName, nil)
     }).
     Append(http.DefaultClient.Do).
     Append(func(resp *http.Response) (io.Reader, error) {
+	return resp.Body, pipe.CheckErrorf(resp.StatusCode != http.StatusOK, "fetch from api.github.com failed with status: %s", resp.Status)
     }).
     Append(ioutil.ReadAll).
     Append(func(body []byte) (string, error) {
 	var model struct {
 	    CreatedAt string `json:"created_at"`
 	}
+	err := json.Unmarshal(body, &model)
+	return model.CreatedAt, err
     }).
     Append(func(rawDate string) (time.Time, error) {
+	return time.Parse(time.RFC3339, rawDate)
     })
 
 func UserCreatedDate(ctx context.Context, githubName string) (time.Time, error) {
+    out, err := userCreatedDatePipe.Do(ctx, githubName)
+    var createdDate time.Time
+    if err == nil {
+	createdDate = out[0].(time.Time)
     }
+    return createdDate, err
 }
 
 name := "octocat"
 createdDate, err := UserCreatedDate(context.Background(), name)
 if err != nil {
     panic(err)
 }
 fmt.Println("Account", name, "created on", createdDate.Format(time.Stamp))
 // Output:
 // Account octocat created on Jan 25 18:44:36
```

And now using traditional error handling with 1 success path test:
```diff
 func UserCreatedDate(ctx context.Context, githubName string) (time.Time, error) {
+    req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/users/"+githubName, nil)
+    if err != nil {
-	return time.Time{}, err
     }
+    resp, err := http.DefaultClient.Do(req)
+    if err != nil {
-	return time.Time{}, err
     }
+    if resp.StatusCode != http.StatusOK {
-	return time.Time{}, fmt.Errorf("fetch from api.github.com failed with status: %s", resp.Status)
     }
+    body, err := ioutil.ReadAll(resp.Body)
+    if err != nil {
-	return time.Time{}, err
     }
 
     var model struct {
 	CreatedAt string `json:"created_at"`
     }
+    err = json.Unmarshal(body, &model)
+    if err != nil {
-	return time.Time{}, err
     }
+    return time.Parse(time.RFC3339, rawDate)
 }
 
 name := "octocat"
 createdDate, err := UserCreatedDate(context.Background(), name)
 if err != nil {
     panic(err)
 }
 fmt.Println("Account", name, "created on", createdDate.Format(time.Stamp))
```

The traditional approach using 1 test failed to cover all possible error paths. Normally this would require we write tests for every path to achieve full branch coverage and ensure 100% correctness.

However, pipes can dramatically reduce the test effort and code complexity. The success path test we wrote covered every possible branch -- our code doesn't have any branches left to cover! *

`Pipe`s simplify and test your code while keeping the original intent clear. Implemented well, they could completely eliminate whole classes of potential bugs.

Questions or concerns? [Let us know what you think.](https://github.com/JohnStarich/go/issues/new)

_* To be completely thorough, we may also want an error path test to cover the custom error message return value. That is left up to the reader._

