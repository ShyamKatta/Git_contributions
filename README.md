# Git_contributions
The below file give the git contributions of a user given a valid access_token and username

Need to replace the access_token in line 228 of main function and username wherever applciable

The go function can replace file - https://github.com/ShyamKatta/Aporeto_challenge/blob/master/scripts/MyContributions_public.py and making necessary changes to nodejs script can result in same output.

The change was conversion of python code to similar code in Golang, but instead of JSON api calls and parsing them, used go-github library

  // Eric's general NOTE's:

	// It is always a good idea to have a function have a return with an error so you can check if it failed
  
	// Any errors that you get back should either be propagated to the top and/or displayed for debuggability
  
	// Try to use channels for concurrency resolution or exploitation, otherwise the code may become overly complex
  
	// Make sure to close all channels once you are done with them, or else they will leak (I do not see a close for `done`)
