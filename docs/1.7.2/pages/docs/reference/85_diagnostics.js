<doc-view>

<h2 id="_diagnostics">Diagnostics</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>There are various commands that allow you to obtain diagnostic output such as
Java Flight Recordings or JFR&#8217;s, heap dumps and thread dumps.</p>

<ul class="ulist">
<li>
<p><router-link to="#get-jfrs" @click.native="this.scrollFix('#get-jfrs')"><code>cohctl get jfrs</code></router-link> - display the JFR&#8217;s for a cluster</p>

</li>
<li>
<p><router-link to="#start-jfr" @click.native="this.scrollFix('#start-jfr')"><code>cohctl start jfr</code></router-link> - start a JFR for all or selected members</p>

</li>
<li>
<p><router-link to="#describe-jfr" @click.native="this.scrollFix('#describe-jfr')"><code>cohctl describe jfr</code></router-link> - describe a JFR</p>

</li>
<li>
<p><router-link to="#stop-jfr" @click.native="this.scrollFix('#stop-jfr')"><code>cohctl stop jfr</code></router-link> - stop a JFR for all or selected members</p>

</li>
<li>
<p><router-link to="#dump-jfr" @click.native="this.scrollFix('#dump-jfr')"><code>cohctl dump jfr</code></router-link> - dump a JFR that is running for all or selected members</p>

</li>
<li>
<p><router-link to="#dump-cluster-heap" @click.native="this.scrollFix('#dump-cluster-heap')"><code>cohctl dump cluster-heap</code></router-link> - dumps the cluster heap for all or specific roles</p>

</li>
<li>
<p><router-link to="#log-cluster-state" @click.native="this.scrollFix('#log-cluster-state')"><code>cohctl log cluster-state</code></router-link> - logs the cluster state via, a thread dump, for all or specific roles</p>

</li>
<li>
<p><router-link to="#retrieve-thread-dumps" @click.native="this.scrollFix('#retrieve-thread-dumps')"><code>cohctl retrieve thread-dumps</code></router-link> - retrieves thread dumps for all or specific nodes</p>

</li>
<li>
<p><router-link to="#configure-tracing" @click.native="this.scrollFix('#configure-tracing')"><code>cohctl configure tracing</code></router-link> - configures tracing for all members or a specific role</p>

</li>
<li>
<p><router-link to="#get-tracing" @click.native="this.scrollFix('#get-tracing')"><code>cohctl get tracing</code></router-link> - displays tracing status for all members</p>

</li>
<li>
<p><router-link to="#get-environment" @click.native="this.scrollFix('#get-environment')"><code>cohctl get environment</code></router-link> - displays the environment for a member</p>

</li>
</ul>

<h4 id="get-jfrs">Get JFRS</h4>
<div class="section">
<p>The 'get jfrs' command displays the Java Flight Recordings for a cluster.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get jfrs [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for jfrs</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl get jfrs -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Member 2-&gt;
    Recording 12: name=test-1 duration=1m (running)
Member 3-&gt;
    Recording 12: name=test-1 duration=1m (running)
Member 4-&gt;
    Recording 12: name=test-1 duration=1m (running)
Member 6-&gt;
    Recording 12: name=test-1 duration=1m (running)</markup>

</div>

<h4 id="start-jfr">Start JFR</h4>
<div class="section">
<p>The 'start jfr' command starts a Java Flight Recording all or selected members.
You can specify either a node id or role. If you do not specify either, then the JFR will
be run for all members. The default duration is 60 seconds and you can specify a value
of 0 to make the recording continuous.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl start jfr name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -D, --duration int32         duration for JFR in seconds. Use 0 for continuous (default 60)
  -h, --help                   help for jfr
  -n, --node string            node id to target
  -O, --output-dir string      directory on servers to output JFR's to
  -r, --role string            role name to target (default "all")
  -s, --settings-file string   settings file to use, options are "profile" or a specific file (default "default")
  -y, --yes                    automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<p>Start a JFR for all members using the defaults (60 seconds duration) and write the results to the <code>/tmp</code> directory on each of the
servers running Coherence members.</p>

<div class="admonition note">
<p class="admonition-inline">If you wish to continuously run a Flight Recording, then set the duration to 0 by using <code>-D 0</code>.</p>
</div>
<markup
lang="bash"

>cohctl start jfr test-1 -O /tmp/ -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to start a JFR named test-1 for all 4 members of duration: 60 seconds? (y/n) y
Member 2-&gt;
  Started recording 11. The result will be written to:
  /tmp/2-test-1.jfr
Member 3-&gt;
  Started recording 11. The result will be written to:
  /tmp/3-test-1.jfr
Member 4-&gt;
  Started recording 11. The result will be written to:
  /tmp/4-test-1.jfr
Member 6-&gt;
  Started recording 11. The result will be written to:
  /tmp/6-test-1.jfr</markup>

<p>Start a JFR for an individual node.</p>

<markup
lang="bash"

>cohctl start jfr test-1 -O /tmp/ -n 2 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to start a JFR named test-1 for node id 2 of duration: 60 seconds? (y/n) y
Started recording 13.

Use jcmd 11339 JFR.stop name=test-1 to copy recording data to file.</markup>

<p>Start a JFR for all members of a specific role.</p>

<markup
lang="bash"

>cohctl start jfr test-1 -O /tmp/ -r CoherenceServer -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to start a JFR named test-1 for role CoherenceServer of duration: 60 seconds? (y/n) y
Member 2-&gt;
  Started recording 14. The result will be written to:
  /tmp/2-test-1.jfr
Member 3-&gt;
  Started recording 13. The result will be written to:
  /tmp/3-test-1.jfr</markup>

</div>

<h4 id="describe-jfr">Describe JFR</h4>
<div class="section">
<p>The 'describe jfr' command shows information related to a Java Flight Recording (JFR).</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl describe jfr name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help          help for jfr
  -n, --node string   node id to target</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl describe jfr test-1 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Member 2-&gt;
    Recording 12: name=test-1 duration=1m (running)
Member 3-&gt;
    Recording 12: name=test-1 duration=1m (running)
Member 4-&gt;
    Recording 12: name=test-1 duration=1m (running)
Member 6-&gt;
    Recording 12: name=test-1 duration=1m (running)</markup>

</div>

<h4 id="stop-jfr">Stop JFR</h4>
<div class="section">
<p>The 'stop jfr' command stops a Java Flight Recording all or selected members.
You can specify either a node or leave the node blank to stop for all nodes.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl stop jfr name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help          help for jfr
  -n, --node string   node id to target
  -y, --yes           automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl stop jfr test1 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to run jfrStop on a JFR named test-1 for all 4 members ? (y/n) y
Member 2-&gt;
    Can't stop an already stopped recording.
Member 3-&gt;
    Stopped recording "test-1".
Member 4-&gt;
    Stopped recording "test-1".
Member 6-&gt;
    Stopped recording "test-1".</markup>

</div>

<h4 id="dump-jfr">Dump JFR</h4>
<div class="section">
<p>The 'dump jfr' command dumps a Java Flight Recording all or selected members.
A JFR command mut be in progress for this to succeed.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl dump jfr name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -f, --filename string   filename for jfr dump
  -h, --help              help for jfr
  -n, --node string       node id to target
  -y, --yes               automatically confirm the operation</pre>
</div>

<p>Normally when a Flight Recording has been finished it will be dump to the output file. If you want to
dump the JFR before it has completed, then you can use this command.</p>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl dump jfr test1 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to run jfrDump on a JFR named test-1 for all 4 members ? (y/n) y
Member 2-&gt;
    Dumped recording "test-1", 590.9 kB written to:
    /tmp/hotspot-pid-11339-id-13-2021_11_01_10_15_35.jfr
Member 3-&gt;
    Dumped recording "test-1", 420.2 kB written to:
    /private/tmp/3-test-1.jfr
Member 4-&gt;
    Dumped recording "test-1", 383.4 kB written to:
    /private/tmp/4-test-1.jfr
Member 6-&gt;
    Dumped recording "test-1", 466.1 kB written to:
    /private/tmp/6-test-1.jfr</markup>

</div>

<h4 id="dump-cluster-heap">Dump Cluster Heap</h4>
<div class="section">
<p>The 'dump cluster-heap' command issues a heap dump for all members or the selected role
by using the -r flag.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl dump cluster-heap [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help          help for cluster-heap
  -r, --role string   role name to run for (default "all")
  -y, --yes           automatically confirm the operation</pre>
</div>

<div class="admonition note">
<p class="admonition-inline">Depending upon your Java heap size and usage, this command can create large files on your temporary file system.</p>
</div>
<p><strong>Examples</strong></p>

<p>Dump cluster heap for all members.</p>

<markup
lang="bash"

>cohctl dump cluster-heap -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to dump cluster heap for all 3 members? (y/n) y
Operation completed. Please see cache server log file for more information</markup>

<p>Dump cluster heap for a specific role.</p>

<markup
lang="bash"

>cohctl dump cluster-heap -c local -r TangosolNetCoherence</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to dump cluster heap for 2 members with role TangosolNetCoherence? (y/n) y
Operation completed. Please see cache server log file for more information</markup>

<div class="admonition note">
<p class="admonition-inline">View the Coherence log files for the location and names of the heap dumps.</p>
</div>
</div>

<h4 id="log-cluster-state">Log Cluster State</h4>
<div class="section">
<p>The 'log cluster-state' command logs a full thread dump and outstanding
polls, in the logs files, for all members or the selected role by using the -r flag.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl log cluster-state [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help          help for cluster-state
  -r, --role string   role name to run for (default "all")
  -y, --yes           automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<p>Log cluster state for all members into the cache server log files.</p>

<markup
lang="bash"

>cohctl log cluster-state -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to log cluster state for all 3 members? (y/n) y
Operation completed. Please see cache server log file for more information</markup>

<p>Log cluster state for a specific role into the cache server log files.</p>

<markup
lang="bash"

>cohctl log cluster-state -c local -r TangosolNetCoherence</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to log cluster state for 2 members with role TangosolNetCoherence? (y/n) y
Operation completed. Please see cache server log file for more information</markup>

</div>

<h4 id="retrieve-thread-dumps">Retrieve Thread Dumps</h4>
<div class="section">
<p>The 'get thead-dumps' command generates and retrieves thread dumps for all or selected
members and places them in the specified (local) output directory. You are also able to specify
a role to retrieve thread dumps for.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl retrieve thread-dumps [node-ids] [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -D, --dump-delay int32    delay between each thread dump (default 10)
  -h, --help                help for thread-dumps
  -n, --number int32        number of thread dumps to retrieve (default 5)
  -O, --output-dir string   existing local directory to output thread dumps to
  -r, --role string         role name to run for (default "all")
  -v, --verbose             produces verbose output
  -y, --yes                 automatically confirm the operation</pre>
</div>

<p>When taking thread dumps, if you want full deadlock analysis, set the following system property
on your Coherence JVM&#8217;s:</p>

<ul class="ulist">
<li>
<p>12.2.1.4.x:     <code>-Dcom.oracle.common.util.Threads.dumpLocks=FULL</code></p>

</li>
<li>
<p>Later versions: <code>-Dcom.oracle.coherence.common.util.Threads.dumpLocks=FULL</code></p>

</li>
</ul>
<p><strong>Examples</strong></p>

<p>Retrieve thread dumps using the defaults of 5 thread dumps each 10 seconds for all members and place them in the <code>/tmp/</code> directory.</p>

<markup
lang="bash"

>cohctl retrieve thread-dumps -O /tmp all -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>This operation will take at least 40 seconds.
Are you sure you want to retrieve 5 thread dumps, each 10 seconds apart for 4 nodes? (y/n) y
Completed 5 of 5 (100.00%)All thread dumps completed and written to /tmp

$ ls -l /tmp/thread-dump-node-*
-rw-r--r--  1 user  wheel  42507 27 Oct 14:44 /tmp/thread-dump-node-1-1.log
-rw-r--r--  1 user  wheel  45422 27 Oct 14:44 /tmp/thread-dump-node-1-2.log
-rw-r--r--  1 user  wheel  45299 27 Oct 14:45 /tmp/thread-dump-node-1-3.log
-rw-r--r--  1 user  wheel  45299 27 Oct 14:45 /tmp/thread-dump-node-1-4.log
-rw-r--r--  1 user  wheel  45311 27 Oct 14:45 /tmp/thread-dump-node-1-5.log
-rw-r--r--  1 user  wheel  35515 27 Oct 14:44 /tmp/thread-dump-node-2-1.log
-rw-r--r--  1 user  wheel  35503 27 Oct 14:44 /tmp/thread-dump-node-2-2.log
-rw-r--r--  1 user  wheel  35503 27 Oct 14:45 /tmp/thread-dump-node-2-3.log
-rw-r--r--  1 user  wheel  35503 27 Oct 14:45 /tmp/thread-dump-node-2-4.log
-rw-r--r--  1 user  wheel  35491 27 Oct 14:45 /tmp/thread-dump-node-2-5.log
-rw-r--r--  1 user  wheel  31579 27 Oct 14:44 /tmp/thread-dump-node-3-1.log
-rw-r--r--  1 user  wheel  31591 27 Oct 14:44 /tmp/thread-dump-node-3-2.log
-rw-r--r--  1 user  wheel  31579 27 Oct 14:45 /tmp/thread-dump-node-3-3.log
-rw-r--r--  1 user  wheel  31591 27 Oct 14:45 /tmp/thread-dump-node-3-4.log
-rw-r--r--  1 user  wheel  31591 27 Oct 14:45 /tmp/thread-dump-node-3-5.log
-rw-r--r--  1 user  wheel  31587 27 Oct 14:44 /tmp/thread-dump-node-4-1.log
-rw-r--r--  1 user  wheel  31587 27 Oct 14:44 /tmp/thread-dump-node-4-2.log
-rw-r--r--  1 user  wheel  31587 27 Oct 14:45 /tmp/thread-dump-node-4-3.log
-rw-r--r--  1 user  wheel  31587 27 Oct 14:45 /tmp/thread-dump-node-4-4.log
-rw-r--r--  1 user  wheel  31587 27 Oct 14:45 /tmp/thread-dump-node-4-5.log</markup>

<div class="admonition note">
<p class="admonition-inline">The files will be named <code>thread-dump-node-N-I.log</code>. Where <code>N</code> is the node id, and <code>I</code> is the iteration.</p>
</div>
<p>Retrieve 5 thread dumps for members 1 and 3 every 15 seconds and place them in the <code>/tmp/</code> directory.</p>

<markup
lang="bash"

>cohctl retrieve thread-dumps -O /tmp 1,3 -n 5 -D 15 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>This operation will take at least 60 seconds.
Are you sure you want to retrieve 5 thread dumps, each 15 seconds apart for 2 nodes? (y/n) y
Completed 5 of 5 (100.00%)
All thread dumps completed and written to /tmp</markup>

<p>Retrieve thread dumps for a given role:</p>

<markup
lang="bash"

>cohctl retrieve thread-dumps  -O /tmp/ -r TangosolNetCoherence -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>This operation will take at least 40 seconds.
Are you sure you want to retrieve 5 thread dumps, each 10 seconds apart for 2 node(s)? (y/n)</markup>

</div>

<h4 id="configure-tracing">Configure Tracing</h4>
<div class="section">
<p>The 'configure tracing' command configures tracing for all members or the selected role
by using the -r flag. You can specify a tracingRatio of -1 to turn off tracing.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl configure tracing [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help                   help for tracing
  -r, --role string            role name to configure tracing for
  -t, --tracingRatio float32   tracing ratio to set. -1.0 turns off tracing (default 1)
  -y, --yes                    automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<p>Configure tracing for all members with tracing ratio of 0.</p>

<markup
lang="bash"

>cohctl configure tracing -t 0 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to configure tracing to tracing ratio 0 for all 3 members? (y/n) y
Operation completed. Please see cache server log file for more information</markup>

<p>Configure tracing for a specific role with tracing ratio of 1.0</p>

<markup
lang="bash"

>cohctl configure tracing -t 1.0 -r TangosolNetCoherence -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to configure tracing to tracing ratio 1 for 2 members with role TangosolNetCoherence? (y/n) y
Operation completed. Please see cache server log file for more information</markup>

<p>Turn off tracing for all members by setting tracing ratio to -1.0.</p>

<markup
lang="bash"

>cohctl configure tracing -t -1.0</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to configure tracing to tracing ratio -1 for all 3 members? (y/n) y
Operation completed. Please see cache server log file for more information</markup>

</div>

<h4 id="get-tracing">Get Tracing</h4>
<div class="section">
<p>The 'get tracing' command displays tracing status for all members.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get tracing [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for tracing</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display tracing for all members.</p>

<markup
lang="bash"

>cohctl get tracing -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>NODE ID  ADDRESS      PORT  PROCESS  MEMBER  ROLE                TRACING ENABLED  SAMPLING RATIO
      1  /127.0.0.1  62255    13464  n/a     DefaultCacheServer  true                      1.000</markup>

</div>

<h4 id="get-environment">Get Environment</h4>
<div class="section">
<p>The 'get environment' command returns the environment information for a member.
This includes details of the JVM as well as system properties.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get environment node-id [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for environment</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display the member environment for a member.</p>

<markup
lang="bash"

>cohctl get environment 1 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Java Version: 11.0.10
Java Vendor:
 - Name: Oracle Corporation
 - Version: 18.9
Java Virtual Machine:
...</markup>

<div class="admonition note">
<p class="admonition-inline">Output has been truncated above.</p>
</div>
</div>
</div>

<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/reference/15_members">Members</router-link></p>

</li>
<li>
<p><a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/develop-applications/debugging-coherence.html">Distributed Tracing in Coherence</a></p>

</li>
</ul>
</div>
</div>
</doc-view>
