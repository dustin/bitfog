# Bitfog

Let's say you've got several terabytes of storage in a couple
different locations.  In one of these locations, lots of interesting
data is being collected.  In the other, there's just capacity.
Connectivity between these two sites falls somewhere between
non-existent, and too slow to matter.  However, you physically travel
between these two sites regularly and have a bit of spare capacity on
your hard drive or something.

This may be the project for you.

# Obvious Questions

## Why Wouldn't I Use Rsync?

rsync's batching mode seems *really* close to my goals, but that's not
its purpose.  It can probably be tweaked to do the right thing, but
it's not entirely obvious how without having the two sites know about
each other.  rdiff came about to solve this problem some, but...

## Why Wouldn't I Use Rdiff?

rdiff has [successfully been used][rd1] to achieve a similar goal, but
still not quite perfectly enough for my needs.  It's just a bit clunky
to use even with these scripts, and when it doesn't work, doesn't work
quite hard.

In my case, I needed a network service primarily, though.  I *can* use
file services to achieve the same goal, of course, but standardizing
on HTTP makes it easier to know what's there and what it's doing.

I am, in general, working with some large files that I may end up
grabbing incrementally, so rdiff is in the roadmap, but I've got more
quantity at the moment.

# Usage

First, to run a server, you need to create a file map.  By default,
this is called `bitfog.json`.  Here's an example:

    {
        "vms": {"path": "/bigpool/vm_images/", "writable": false}
    }

This makes a read-only view of `/bigpool/vm_images` serviced as
`vms`.  This makes it a "source" only as far as bitfog is concerned.
For something to be a suitable destination, you just set "writable" to
`true`.

Next, you build a DB describing the things found in that source:

    bitfog builddb http://myserver:8675/vms/ vms.db

For syncing to the remote end, you'll want a similar DB that
represents the state on that server.  You'd use the same procedure as
above to build the db, or you can use the `emptydb` command to build a
database that says there's nothing on the remote end.

Now, grab some of the missing data:

    mkdir ~/tmp/bitfog.tmp
    bitfog fetch dest.db http://myserver:8675/vms/ ~/tmp/bitfog.tmp

This uses the `dest.db` we created in that represents the destination
(possibly empty) server and asks the source for anything that's
missing, holding it temporarily in `~/tmp/bitfog.tmp`.

Once we figured out we've got enough, or it's time to go to the other
location, we stop, pack up, get on the train, and wait for our arrival
at the new location.  Once there, we can see our other bitfog
instance, and we feed it some of our data:

    bitfog store vms.db http://emptyserver:8675/vms/ ~/tmp/bitfog.tmp

This begins to fill the server up with some of our temporary data we
fetched.

Don't forget to run `builddb` again when you're done so we can get a
snapshot of the current state before going back to the other site to
start moving more data.


[rd1]: http://users.softlab.ece.ntua.gr/~ttsiod/Offline-rsync.html
