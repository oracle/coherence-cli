<doc-view>

<h2 id="_elastic_data">Elastic Data</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>There are various commands that allow you to work with and manage Elastic Data.</p>

<ul class="ulist">
<li>
<p><router-link to="#get-elastic-data" @click.native="this.scrollFix('#get-elastic-data')"><code>cohctl get elastic-data</code></router-link> - displays the elastic data details</p>

</li>
<li>
<p><router-link to="#describe-elastic-data" @click.native="this.scrollFix('#describe-elastic-data')"><code>cohctl describe elastic-data</code></router-link> - shows information related to a specific journal type</p>

</li>
<li>
<p><router-link to="#compact-elastic-data" @click.native="this.scrollFix('#compact-elastic-data')"><code>cohctl compact elastic-data</code></router-link> - compacts a flash or ram journal</p>

</li>
</ul>
<div class="admonition note">
<p class="admonition-inline">This is a Coherence Grid Edition feature only and is not available with Community Edition.</p>
</div>

<h4 id="get-elastic-data">Get Elastic Data</h4>
<div class="section">
<p>The 'get elastic-data' command displays the Flash Journal and RAM
Journal details for the cluster.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get elastic-data [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for elastic-data</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display elastic data.</p>

<markup
lang="bash"

>cohctl get http-servers -c local -m</markup>

<p>Output:</p>

<markup
lang="bash"

>NAME            USED FILES  TOTAL FILES  % USED  MAX FILE SIZE  USED SPACE   COMMITTED  HIGHEST LOAD  COMPACTIONS  EXHAUSTIVE
RamJournalRM            80       19,600   0.41%           1 MB        0 MB       80 MB        0.0041            0           0
FlashJournalRM          81       41,391   0.20%        2,048 MB       0 MB  162,000 GB        0.0020            0           0</markup>

</div>

<h4 id="describe-elastic-data">Describe Elastic Data</h4>
<div class="section">
<p>The 'describe elastic-data' command shows information related to a specific journal type.
The allowable values are RamJournalRM or FlashJournalRM.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl describe elastic-data {FlashJournalRM|RamJournalRM} [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for elastic-data</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl describe elastic-data RamJournalRM -c local</markup>

</div>

<h4 id="compact-elastic-data">Compact Elastic Data</h4>
<div class="section">
<p>The 'compact elastic-data' command compacts (garbage collects) a specific journal type
for all or specific nodes. The allowable values are RamJournalRM or FlashJournalRM.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl compact elastic-data {FlashJournalRM|RamJournalRM} [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help          help for elastic-data
  -n, --node string   comma separated node ids to target (default "all")
  -y, --yes           automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<p>Compact flash journal for all nodes.</p>

<markup
lang="bash"

>cohctl compact elastic-data FlashJournalRM -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to compact flash for all 2 nodes? (y/n) y
operation completed</markup>

<p>Compact ram journal for 1 node.</p>

<markup
lang="bash"

>cohctl compact elastic-data RamJournalRM -n 1 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to compact ram for 1 node(s)? (y/n) y
operation completed</markup>

</div>
</div>
</div>
</doc-view>
