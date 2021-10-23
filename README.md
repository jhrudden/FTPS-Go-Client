# Project 2: FTPS Client

## My Approach

I implemented this project using Golang. As for the steps in which I implemented each part of my program, I strictly followed the _Suggested Implemenation Process_ outlined on the project description page. I used the net package for golang to create connections to remote host and tls package from crypto for golang to wrap connections with tls handshakes. I am reusing functionality from the past project to read and write to the server. That mostly covered my approach to completing these problems. The caveate is, for both cp and mv commands, Since the whole process was not described in the project description, I mostly thought those up on the stop. For mv and cp, I use both the io and os packages for golang to open and create files on my local machine and copy contents either to the remote or from the remote into new files. If I am moving contents from the remote to local, then I then make a DELE call to the remote. Otherwise, if I am moving a local file to remote, I use the before explained package to remove it said file. With that the project was done.

## Challenges

My two largest issues dealt with creating buffers for multiline responses from servers and calling functions that required close data connection without closing data connection.

For the first issue, I struggled to get more than one line returned after calling ls command, I finally figured that I was dumping the messages into a reader and since I was only reading till \n, I was missing multiline responses. I created my **readAll()** function to solve this issue.

For my second issue, I continually struggled to delete files after copying them from remote to local for _mv_ function. Since I had not closed my data channel, the server was expecting data requests, so making a dele call didn't seem to be working. Once I stumbled upon this realization, closing the data channel imediately allowed me to delete files.

## Testings Process

The majority of my testings was done through print statements and trial and error through using the servers responses. I referred to definitions of error codes given in the project description for understanding what issues my program was facing.
