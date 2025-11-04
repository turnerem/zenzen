## Go Log

Go and log your work. A personal log written in Go.

A log of achievements, and an account of how little time it was thought it would take to complete them. 
Log initial predictions of how long it will take to complete a piece of work. 
Then return at the end to log how long it actually took. Build up a graph of optimism over time.
Perhaps correct for this bias over time.

Nevertheless, this will be a log of completed work, useful for justifying ones value to employers.

Terminal-based log with local file storage.

List logs:
```
```

Add log:
```
```

Edit log:
```
```

Delete log:
```
```

Display tags (and counts):
```
```

Chart diff of predicted versus actual time to completion:
```
```

### How it works

Everything is done in the terminal. To start a new log, run `gli add <TITLE>` in the command line. TITLE must be unique in the directory. This will open a new doc that can be navigated with vim commands. The doc follows a template:

```
Title: TITLE

Tags: describe categories of work, etc

TTC Prediction: hours/days/weeks

TTC Actual: to be added once work is completed

And this is the body of the log. It can be blank, but ideally it should contain a description of what was done, what was achieved, obstacles overcome. It can include notes on how a problem can be better dealt with next time, and general learnings.
```

The program is written in Go. The log will look pretty in the terminal. The correct tools from charmbracelet must be chosen for this purpose. When logs are saved, it should be as a JSON file per log. The structure of the JSON is

```
{
    "Title": string,
    "Tags": []string,
    "TTC Prediction": string,
    "TTC Actual": string,
    "Body": string,
}
```

Note that both TTC measurements are `string` and must be converted to the correct unit by the program. A separate file of tags per TITLE is kept. It is updated everytime a log is created, updated or deleted, as required. This must happen in the same transaction.