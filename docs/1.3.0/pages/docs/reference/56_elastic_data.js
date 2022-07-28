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

>$ cohctl get http-servers -c local -m

NAME            USED FILES  TOTAL FILES  % USED  MAX FILE SIZE  USED SPACE   COMMITTED  HIGHEST LOAD  COMPACTIONS  EXHAUSTIVE
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

>$ cohctl describe elastic-data RamJournalRM -c local</markup>

</div>
</div>
</div>
</doc-view>
