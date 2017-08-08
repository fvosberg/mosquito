

## Questions

0. Should I export the handlers to test them from an todo_test package to remove
   the tests from my library, or should I leave them in the package to not
   pollute the package API?
0. Should I add the user ID in authenticated to the context to have clean
   http.Handler/http.HandlerFunc or should I leave it like it is, to have the
   exact dependency and don't have to check for it's existence/type on runtime?
0. Should I remove the error return value from authenticated for a cleaner
   decorator API and a more "nil value philosophy" and just deny all requests,
   which are secured?
