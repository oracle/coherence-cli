<doc-view>

<h2 id="_topics">Topics</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>There are various commands that allow you to work with and manage cluster topics.</p>

<ul class="ulist">
<li>
<p><router-link to="#get-topics" @click.native="this.scrollFix('#get-topics')"><code>cohctl get topics</code></router-link> - displays topics for a cluster</p>

</li>
<li>
<p><router-link to="#describe-topic" @click.native="this.scrollFix('#describe-topic')"><code>cohctl describe topic</code></router-link> - describe a topic</p>

</li>
<li>
<p><router-link to="#get-topic-members" @click.native="this.scrollFix('#get-topic-members')"><code>cohctl get topic-members</code></router-link> - displays members for a topic</p>

</li>
<li>
<p><router-link to="#get-topic-channels" @click.native="this.scrollFix('#get-topic-channels')"><code>cohctl get topic-channels</code></router-link> - displays channel details for a topic, service and node</p>

</li>
<li>
<p><router-link to="#get-subscribers" @click.native="this.scrollFix('#get-subscribers')"><code>cohctl get subscribers</code></router-link> - displays subscribers for a topic and service</p>

</li>
<li>
<p><router-link to="#get-subscriber-channels" @click.native="this.scrollFix('#get-subscriber-channels')"><code>cohctl get subscriber-channels</code></router-link> - displays channel details for a topic, service and subscriber</p>

</li>
<li>
<p><router-link to="#get-subscriber-groups" @click.native="this.scrollFix('#get-subscriber-groups')"><code>cohctl get subscriber-groups</code></router-link> - displays subscriber-groups for a topic and service</p>

</li>
<li>
<p><router-link to="#get-sub-grp-channels" @click.native="this.scrollFix('#get-sub-grp-channels')"><code>cohctl get sub-grp-channels</code></router-link> - displays channel details for a topic, service, node and subscriber group</p>

</li>
<li>
<p><router-link to="#disconnect-all" @click.native="this.scrollFix('#disconnect-all')"><code>cohctl disconnect all</code></router-link> - instructs a topic to disconnect all subscribers for a topic or subscriber group</p>

</li>
</ul>

<p>Subscriber Specific Operations</p>

<ul class="ulist">
<li>
<p><router-link to="#connect-subscriber" @click.native="this.scrollFix('#connect-subscriber')"><code>cohctl connect subscriber</code></router-link> - instructs a subscriber to ensure it is connected</p>

</li>
<li>
<p><router-link to="#disconnect-subscriber" @click.native="this.scrollFix('#disconnect-subscriber')"><code>cohctl disconnect subscriber</code></router-link> - instructs a subscriber to disconnect and reset itself</p>

</li>
<li>
<p><router-link to="#retrieve-heads" @click.native="this.scrollFix('#retrieve-heads')"><code>cohctl retrieve heads</code></router-link> - instructs a subscriber to retrieve the current head positions for each channel</p>

</li>
<li>
<p><router-link to="#retrieve-remaining" @click.native="this.scrollFix('#retrieve-remaining')"><code>cohctl retrieve remaining</code></router-link> - instructs a subscriber to retrieve the count of remaining messages for each channel</p>

</li>
<li>
<p><router-link to="#notify-populated" @click.native="this.scrollFix('#notify-populated')"><code>cohctl notify populated</code></router-link> - instructs a subscriber to send a channel populated notification to this subscriber and channel</p>

</li>
</ul>

<div class="admonition note">
<p class="admonition-inline">These topics commands are available to run against Coherence CE editions 22.06.4+, 23.03.+ and 24.03+ as well as
Coherence Grid Edition 14.1.1.2206.4+.</p>
</div>

<div class="admonition note">
<p class="admonition-inline">In most commands you may omit the service name option if the topic name is unique.</p>
</div>


<h4 id="get-topics">Get Topics</h4>
<div class="section">
<p>The 'get topics' command displays topics for a cluster.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get topics [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for topics
  -s, --service string   Service name</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display all topics.</p>

<markup
lang="bash"

>cohctl get topics -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>SERVICE           TOPIC             MEMBERS  CHANNELS  SUBSCRIBERS  PUBLISHED
PartitionedTopic  private-messages        1        17            1          0
PartitionedTopic  public-messages         1        17            1          0</markup>

</div>


<h4 id="describe-topic">Describe Topic</h4>
<div class="section">
<p>The 'describe topic' command shows information related to a topic and service.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl describe topic topic-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for topic
  -s, --service string   Service name</pre>
</div>

<markup
lang="bash"

>cohctl describe topic private-messages -s PartitionedTopic -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>TOPIC DETAILS
-------------
Name           :  private-messages
Service        :  PartitionedTopic
Channels       :  17
Members        :  1
Published Count:  0
Subscribers    :  1

MEMBERS
-------
NODE ID  CHANNELS  PUBLISHED    MEAN   1 MIN   5 MIN  15 MIN
      1        17          0  0.0000  0.0000  0.0000  0.0000

SUBSCRIBERS
-----------
NODE ID  SUBSCRIBER ID  STATE      CHANNELS  SUBSCRIBER GROUP  RECEIVED  ERRORS  BACKLOG
      1     5850459679  Connected        17  1                        0       0        0

SUBSCRIBER GROUPS
-----------------
SUBSCRIBER GROUP  NODE ID  CHANNELS  POLLED    MEAN   1 MIN   5 MIN  15 MIN
1                       1        17       0  0.0000  0.0000  0.0000  0.0000
1                       2        17       0  0.0000  0.0000  0.0000  0.0000
admin                   1        17       0  0.0000  0.0000  0.0000  0.0000
admin                   2        17       0  0.0000  0.0000  0.0000  0.0000</markup>

</div>


<h4 id="get-topic-members">Get Topic Members</h4>
<div class="section">
<p>The 'get topic-members' command displays members for topic and service.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get topic-members topic-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for topic-members
  -s, --service string   Service name</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display all members for a topic using the wide option</p>

<markup
lang="bash"

>cohctl get topic-members private-messages -s PartitionedTopic -o wide -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Service:  PartitionedTopic
Topic:    private-messages

NODE ID  CHANNELS  PUBLISHED    MEAN   1 MIN   5 MIN  15 MIN  SUB TIMEOUT  RECON TIMEOUT      WAIT  PAGE CAPACITY
      1        17          0  0.0000  0.0000  0.0000  0.0000    300,000ms      300,000ms  10,000ms      1,048,576
      2        17          0  0.0000  0.0000  0.0000  0.0000    300,000ms      300,000ms  10,000ms      1,048,576</markup>

</div>


<h4 id="get-topic-channels">Get Topic Channels</h4>
<div class="section">
<p>The 'get topic-channels' command displays channel details for a topic, service and node.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get topic-channels topic-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for topic-channels
  -n, --node int32       node id to show channels for
  -s, --service string   Service name</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display the channels for a topic, service and node.</p>

<markup
lang="bash"

>cohctl get topic-members private-messages -s PartitionedTopic -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Service:  PartitionedTopic
Topic:    private-messages

NODE ID  CHANNELS  PUBLISHED    MEAN   1 MIN   5 MIN  15 MIN
      4        17          0  0.0000  0.0000  0.0000  0.0000
      5        17          0  0.0000  0.0000  0.0000  0.0000
      6        17          0  0.0000  0.0000  0.0000  0.0000</markup>

<markup
lang="bash"

>cohctl get topic-channels private-messages -s PartitionedTopic -n 4 -c local

Service:      PartitionedTopic
Topic:        private-messages
Node ID:      4
ChannelCount: 17

CHANNEL  PUBLISHED    MEAN   1 MIN   5 MIN  15 MIN  TAIL
      0          0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=-1, offset=2147483647)
      1          0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=-1, offset=2147483647)
      2          0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=-1, offset=2147483647)
      3          0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=-1, offset=2147483647)
      4          0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=-1, offset=2147483647)
      5          0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=-1, offset=2147483647)
      6          0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=-1, offset=2147483647)
      7          0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=-1, offset=2147483647)
      8          0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=-1, offset=2147483647)
      9          0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=-1, offset=2147483647)
     10          0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=-1, offset=2147483647)
     11          0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=-1, offset=2147483647)
     12          0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=-1, offset=2147483647)
     13          0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=-1, offset=2147483647)
     14          0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=-1, offset=2147483647)
     15          0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=-1, offset=2147483647)
     16          0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=-1, offset=2147483647)</markup>

</div>


<h4 id="get-subscribers">Get Subscribers</h4>
<div class="section">
<p>The 'get subscribers' command displays subscribers for a topic and service.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get subscribers topic-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for subscribers
  -s, --service string   Service name</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display the subscribers for a topic and service.</p>

<markup
lang="bash"

>cohctl get subscribers private-messages -s PartitionedTopic -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Service:  PartitionedTopic
Topic:    private-messages

NODE ID  SUBSCRIBER ID  STATE      CHANNELS  SUBSCRIBER GROUP  RECEIVED  ERRORS  BACKLOG  TYPE     OWNED CHANNELS
      4    17663538530  Connected      9/17  1                        0       0        0  Durable  0,1,2,3,4,5,6,7,16
      5    23267662748  Connected      8/17  1                        0       0        0  Durable  8,9,10,11,12,13,14,15</markup>

</div>


<h4 id="get-subscriber-channels">Get Subscriber Channels</h4>
<div class="section">
<p>The 'get subscriber-channels' command displays channel details for a topic, service and subscriber.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get subscriber-channels topic-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for subscriber-channels
  -s, --service string   Service name
  -S, --subscriber int   subscriber id</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display the subscribers channels for a topic, service and subscriber.</p>

<markup
lang="bash"

>cohctl get subscribers private-messages -s PartitionedTopic -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Service:  PartitionedTopic
Topic:    private-messages

NODE ID  SUBSCRIBER ID  STATE      CHANNELS  SUBSCRIBER GROUP  RECEIVED  ERRORS  BACKLOG
      4    17663538530  Connected        17  1                        0       0        0
      5    23267662748  Connected        17  1                        0       0        0
      6    26845397584  Connected        17  1                        0       0        0</markup>

<markup
lang="bash"

>cohctl get subscriber-channels  private-messages -s PartitionedTopic -S 17663538530 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Service:          PartitionedTopic
Topic:            private-messages
Node ID:          4
ChannelCount:     17
Subscriber Group: 17663538530

CHANNEL  EMPTY  LAST COMMIT  LAST REC  OWNED  HEAD
      0  true   null         null      true   PagedPosition(page=66, offset=0)
      1  true   null         null      true   PagedPosition(page=66, offset=0)
      2  true   null         null      true   PagedPosition(page=66, offset=0)
      3  true   null         null      true   PagedPosition(page=66, offset=0)
      4  true   null         null      true   PagedPosition(page=66, offset=0)
      5  false  null         null      false  PagedPosition(page=66, offset=0)
      6  false  null         null      false  PagedPosition(page=66, offset=0)
      7  false  null         null      false  PagedPosition(page=66, offset=0)
      8  false  null         null      false  PagedPosition(page=66, offset=0)
      9  false  null         null      false  PagedPosition(page=66, offset=0)
     10  false  null         null      false  PagedPosition(page=66, offset=0)
     11  false  null         null      false  PagedPosition(page=66, offset=0)
     12  false  null         null      false  PagedPosition(page=66, offset=0)
     13  false  null         null      false  PagedPosition(page=66, offset=0)
     14  false  null         null      false  PagedPosition(page=66, offset=0)
     15  true   null         null      true   PagedPosition(page=66, offset=0)
     16  false  null         null      false  PagedPosition(page=66, offset=0)</markup>

</div>


<h4 id="get-subscriber-groups">Get Subscriber Groups</h4>
<div class="section">
<p>The 'get subscribers' command displays subscriber-groups for a topic and service.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get subscriber-groups topic-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for subscriber-groups
  -s, --service string   Service name</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display the subscriber groups for a topic and service.</p>

<markup
lang="bash"

>cohctl get subscriber-groups private-messages -s PartitionedTopic -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Service:  PartitionedTopic
Topic:    private-messages

SUBSCRIBER GROUP  NODE ID  CHANNELS  POLLED    MEAN   1 MIN   5 MIN  15 MIN
1                       1        17       0  0.0000  0.0000  0.0000  0.0000
1                       2        17       0  0.0000  0.0000  0.0000  0.0000
1                       3        17       0  0.0000  0.0000  0.0000  0.0000
admin                   1        17       0  0.0000  0.0000  0.0000  0.0000
admin                   2        17       0  0.0000  0.0000  0.0000  0.0000
admin                   3        17       0  0.0000  0.0000  0.0000  0.0000</markup>

</div>


<h4 id="get-sub-grp-channels">Get Subscriber Group Channels</h4>
<div class="section">
<p>The 'get sub-grp-channels' command displays channel details for a topic, service, node and subscriber group.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get sub-grp-channels topic-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help                      help for sub-grp-channels
  -n, --node int32                node id to show channels for
  -s, --service string            Service name
  -G, --subscriber-group string   subscriber group</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display subscriber group channels a topic, service, node and subscriber group.</p>

<markup
lang="bash"

>cohctl get sub-grp-channels private-messages -s PartitionedTopic -n 3 -G admin -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Service:          PartitionedTopic
Topic:            private-messages
Node ID:          3
ChannelCount:     17
Subscriber Group: admin

CHANNEL  OWNING SUB  MEMBER  POLLED    MEAN   1 MIN   5 MIN  15 MIN  HEAD
      0          -1      -1       0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=0, offset=0)
      1          -1      -1       0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=0, offset=0)
      2          -1      -1       0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=0, offset=0)
      3          -1      -1       0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=0, offset=0)
      4          -1      -1       0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=0, offset=0)
      5          -1      -1       0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=0, offset=0)
      6          -1      -1       0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=0, offset=0)
      7          -1      -1       0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=0, offset=0)
      8          -1      -1       0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=0, offset=0)
      9          -1      -1       0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=0, offset=0)
     10          -1      -1       0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=0, offset=0)
     11          -1      -1       0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=0, offset=0)
     12          -1      -1       0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=0, offset=0)
     13          -1      -1       0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=0, offset=0)
     14          -1      -1       0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=0, offset=0)
     15          -1      -1       0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=0, offset=0)
     16          -1      -1       0  0.0000  0.0000  0.0000  0.0000  PagedPosition(page=0, offset=0)</markup>

</div>


<h4 id="disconnect-all">Disconnect All</h4>
<div class="section">
<p>The 'disconnect all' command instructs a topic to disconnect all subscribers for a
specific subscriber topic or all subscribers for the specified subscriber group.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl disconnect all topic-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help                      help for all
  -s, --service string            Service name
  -G, --subscriber-group string   subscriber group
  -y, --yes                       automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<p>Disconnect all subscribers for the topic 'sms-messages'.</p>

<markup
lang="bash"

>cohctl disconnect all sms-messages -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to issue 'disconnect all' for topic sms-messages and service PartitionedTopic? (y/n) y
operation completed</markup>

<p>Disconnect all subscribers for the topic 'sms-messages' and subscriber group 'sms-processor'.</p>

<markup
lang="bash"

>cohctl disconnect all sms-messages -G sms-processor -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to issue 'disconnect all' for topic sms-messages, service PartitionedTopic and subscriber group sms-processor? (y/n) y
operation completed</markup>

</div>


<h4 id="connect-subscriber">Connect Subscriber</h4>
<div class="section">
<p>The 'connect subscriber' command instructs a subscriber to ensure it is connected
given a topic, service and subscriber id.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl connect subscriber topic-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for subscriber
  -s, --service string   Service name
  -S, --subscriber int   subscriber id
  -y, --yes              automatically confirm the operation</pre>
</div>

<p><strong>Example</strong></p>

<markup
lang="bash"

>cohctl get subscribers public-messages -s PartitionedTopic -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Service:  PartitionedTopic
Topic:    public-messages

NODE ID  SUBSCRIBER ID  STATE      CHANNELS  SUBSCRIBER GROUP  RECEIVED  ERRORS  BACKLOG
      4    17179869184  Connected        17  1                    7,406       0        0</markup>

<markup
lang="bash"

>cohctl connect subscriber public-messages -s PartitionedTopic -S 17179869184 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to issue 'connect' for topic public-messages, service PartitionedTopic and subscriber 17179869184? (y/n) y
operation completed</markup>

</div>


<h4 id="disconnect-subscriber">Disconnects Subscriber</h4>
<div class="section">
<p>The 'disconnect subscriber' command instructs a subscriber to disconnect and reset
itself given a topic, service and subscriber id.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl disconnect subscriber topic-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for subscriber
  -s, --service string   Service name
  -S, --subscriber int   subscriber id
  -y, --yes              automatically confirm the operation</pre>
</div>

<p><strong>Example</strong></p>

<markup
lang="bash"

>cohctl dicconnect subscriber public-messages -s PartitionedTopic -S 17179869184 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to issue 'disconnect' for topic public-messages, service PartitionedTopic and subscriber 17179869184? (y/n) y
operation completed</markup>

</div>


<h4 id="retrieve-heads">Retrieve Heads</h4>
<div class="section">
<p>The 'retrieve heads' command instructs a subscriber to retrieve the current head
positions for each channel given a topic, service and subscriber id.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl retrieve heads topic-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for heads
  -s, --service string   Service name
  -S, --subscriber int   subscriber id
  -y, --yes              automatically confirm the operation</pre>
</div>

<p><strong>Example</strong></p>

<markup
lang="bash"

>cohctl retrieve heads public-messages -s PartitionedTopic -S 17179869184 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to issue 'retrieve current heads' for topic public-messages, service PartitionedTopic and subscriber 17179869184? (y/n) y

Service:    PartitionedTopic
Topic:      public-messages
Subscriber: 17179869184

CHANNEL  POSITION
      0  PagedPosition(page=96, offset=3082)
      1  PagedPosition(page=96, offset=3411)
      2  PagedPosition(page=97, offset=0)
      3  PagedPosition(page=97, offset=0)
      4  PagedPosition(page=97, offset=0)
      5  PagedPosition(page=97, offset=0)
      6  PagedPosition(page=97, offset=0)
      7  PagedPosition(page=97, offset=0)
      8  PagedPosition(page=97, offset=0)
      9  PagedPosition(page=97, offset=0)
     10  PagedPosition(page=97, offset=0)
     11  PagedPosition(page=97, offset=0)
     12  PagedPosition(page=97, offset=0)
     13  PagedPosition(page=96, offset=1953)
     14  PagedPosition(page=96, offset=2037)
     15  PagedPosition(page=96, offset=2405)
     16  PagedPosition(page=96, offset=2625)</markup>

</div>


<h4 id="retrieve-remaining">Retrieve Remaining</h4>
<div class="section">
<p>The 'retrieve header' command instructs a subscriber to retrieve the the count of
remaining messages for each channel given a topic, service and subscriber id.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl retrieve remaining topic-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for remaining
  -s, --service string   Service name
  -S, --subscriber int   subscriber id
  -y, --yes              automatically confirm the operation</pre>
</div>

<p><strong>Example</strong></p>

<markup
lang="bash"

>cohctl retrieve remaining public-messages -s  PartitionedTopic -S 17179869184 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to issue 'retrieve remaining messages' for topic public-messages, service PartitionedTopic and subscriber 17179869184? (y/n) y
operation completed</markup>

</div>


<h4 id="notify-populated">Notify Populated</h4>
<div class="section">
<p>The 'notify populated' command instructs a subscriber to send a channel populated notification to
this subscriber and channel given a topic, service, subscriber id and channel.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl notify populated topic-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -C, --channel int32    channel to notify
  -h, --help             help for populated
  -s, --service string   Service name
  -S, --subscriber int   subscriber id
  -y, --yes              automatically confirm the operation</pre>
</div>

<p><strong>Example</strong></p>

<markup
lang="bash"

>cohctl notify populated public-messages -s PartitionedTopic -S 17179869184 -C 16 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to issue 'notify populated' for topic public-messages, service PartitionedTopic, subscriber 17179869184 and channel 16? (y/n) y
operation completed</markup>

</div>

</div>


<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/reference/caches">Caches</router-link></p>

</li>
<li>
<p><router-link to="/docs/reference/services">Services</router-link></p>

</li>
</ul>

</div>

</div>

</doc-view>
