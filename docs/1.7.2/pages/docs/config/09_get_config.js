<doc-view>

<h2 id="_get_config">Get Config</h2>
<div class="section">
<p>The 'get config' command displays the config stored in the '.cohctl.yaml' config file
in a human readable format.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get config [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help      help for config
  -v, --verbose   include verbose output including cluster connections and profiles</pre>
</div>

<p><strong>Example</strong></p>

<markup
lang="bash"

>cohctl get config</markup>

<p>Output:</p>

<markup
lang="bash"

>CONFIG
------
Version             :  1.7.2
Current Context     :  fp1
Debug               :  true
Color               :  on
Request Timeout     :  30
Ignore Invalid Certs:  false
Default Bytes Format:  m
Default Heap        :  512m
Use Gradle          :  false</markup>

<div class="admonition note">
<p class="admonition-inline">You can use the <code>-v</code> option to display cluster connections and profiles.</p>
</div>
</div>
</doc-view>
