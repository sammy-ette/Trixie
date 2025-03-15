# Trixie
🪻 A simple, pure Go "database" designed specifically to back
[Hilbish](https://github.com/Rosettea/Hilbish)'s history store
in Hilbish v3.

### What?
Trixie is a [Trie-based](https://en.wikipedia.org/wiki/Trie) database store.
The idea is that it'll be highly efficient for autocomplete and history search.
Hilbish currently stores its history in a very simple format: a newline delimited file.
This makes it hard to do anything besides just store the command.

As Hilbish v3 will change the database store, mainly to hold "metadata" about a given
command, a different method to store history was needed. And I was suggested to 
hand roll a Trie-based database, and I thought, why not?

# License
Trixie is MIT licensed!
