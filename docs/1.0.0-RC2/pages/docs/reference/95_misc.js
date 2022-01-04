<doc-view>

<h2 id="_miscellaneous">Miscellaneous</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>This section contains miscellaneous commands for various settings.</p>

<ul class="ulist">
<li>
<p><router-link to="#version" @click.native="this.scrollFix('#version')"><code>cohctl version</code></router-link> - displays the CLI version</p>

</li>
<li>
<p><router-link to="#get-ingore-certs" @click.native="this.scrollFix('#get-ingore-certs')"><code>cohctl get ignore-certs</code></router-link> - displays the current setting for ignoring invalid SSL certificates</p>

</li>
<li>
<p><router-link to="#set-ignore-certs" @click.native="this.scrollFix('#set-ignore-certs')"><code>cohctl set ignore-certs</code></router-link> - sets the current setting for ignoring invalid SSL certificates to true or false</p>

</li>
<li>
<p><router-link to="#get-logs" @click.native="this.scrollFix('#get-logs')"><code>cohctl get logs</code></router-link> - displays the cohctl logs</p>

</li>
<li>
<p><router-link to="#get-debug" @click.native="this.scrollFix('#get-debug')"><code>cohctl set debug</code></router-link> - displays the debug level</p>

</li>
<li>
<p><router-link to="#set-debug" @click.native="this.scrollFix('#set-debug')"><code>cohctl get debug</code></router-link> - sets the debug level on or off</p>

</li>
<li>
<p><router-link to="#get-management" @click.native="this.scrollFix('#get-management')"><code>cohctl get management</code></router-link> - displays management information for a cluster</p>

</li>
<li>
<p><router-link to="#set-management" @click.native="this.scrollFix('#set-management')"><code>cohctl set management</code></router-link> - sets management information for a cluster</p>

</li>
</ul>

<h4 id="version">Version</h4>
<div class="section">
<p>The 'get version' command displays version and build details for the Coherence-CLI.
Use the '-u' option to check for updates.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl version [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -u, --check-updates   If true, will check for updates
  -h, --help            help for version</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl version
Coherence Command Line Interface
CLI Version: 1.0.0
Date:        2021-10-18T02:03:03Z
Commit:      954a008eb87fc9312894d5bbb90edeec8f92bd3a
OS:          darwin
OS Version:  amd64</markup>

<div class="admonition note">
<p class="admonition-inline">You can also use the <code>-u</code> option to check for updates. If you are behind a proxy server, you must also
set the environment variable HTTP_PROXY=http://proxy-host:proxy-port/ so that the update set may be contacted.</p>
</div>
</div>

<h4 id="get-ingore-certs">Get Ignore Certs</h4>
<div class="section">
<p>The 'get ignore-certs' command displays the current setting for ignoring
invalid SSL Certificates. If 'true' then invalid certificates such as self signed will be allowed.
You should only use this option when you are sure of the identify of the target server.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get ignore-certs [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for ignore-certs</pre>
</div>

<div class="admonition note">
<p class="admonition-inline">WARNING: You should only use this option when you are sure of the identity of the target server</p>
</div>
<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl get ignore-certs
Current setting: false</markup>

</div>

<h4 id="set-ignore-certs">Set Ignore Certs</h4>
<div class="section">
<p>The 'set ignore-certs' set the current setting for ignoring
invalid SSL Certificates. If 'true' then invalid certificates such as self signed will be allowed.
You should only use this option when you are sure of the identify of the target server.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl set ignore-certs {true|false} [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for ignore-certs</pre>
</div>

<markup
lang="bash"

>$ cohctl set gnore-certs true
Value is now set to true</markup>

</div>

<h4 id="get-logs">Get Logs</h4>
<div class="section">
<p>The 'get logs' command displays the current contents of the 'cohctl' log file.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get logs [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for logs</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl get logs</markup>

</div>

<h4 id="get-debug">Get Debug</h4>
<div class="section">
<p>The 'get debug' command displays the current debug level. If 'on' then
additional information is logged in the log file.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get debug [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for debug</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl get debug
Current debug level: off</markup>

</div>

<h4 id="set-debug">Set Debug</h4>
<div class="section">
<p>The 'set debug' command sets debug to on or off. If 'on' then additional
information is logged in the log file (cohctl.log).</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl set debug {on|off}} [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for debug</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl set debug on
Debug is now set to on</markup>

</div>

<h4 id="get-management">Get Management</h4>
<div class="section">
<p>The 'get management' command displays the management information for a cluster.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get management [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for management</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl get management -c local

Refresh Policy           :  refresh-behind
Expiry Delay             :  1000
Refresh Count            :  500
Refresh Excess Count     :  143
Refresh On Query         :  false
Refresh Prediction Count :  24389
Refresh Time             :  2021-11-22T03:48:17.739Z
Refresh Timeout Count    :  0
Remote Notification Count:  0
Type                     :  Management</markup>

</div>

<h4 id="set-management">Set Management</h4>
<div class="section">
<p>The 'set management' command sets a management attribute for the cluster.
The following attribute names are allowed: expiryDelay and refreshPolicy.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl set management [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -a, --attribute string   attribute name to set
  -h, --help               help for management
  -v, --value string       attribute value to set
  -y, --yes                Automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<p>Set the expiryDelay to 5000ms for a cluster.</p>

<markup
lang="bash"

>$ cohctl set management -a expiryDelay -v 5000 -c local

Are you sure you want to set the value of attribute expiryDelay to 6000? (y/n) y
operation completed

$ cohctl get management -c local

Refresh Policy           :  refresh-behind
Expiry Delay             :  6000
Refresh Count            :  500
Refresh Excess Count     :  143
Refresh On Query         :  false
Refresh Prediction Count :  24389
Refresh Time             :  2021-11-22T03:50:21.370Z
Refresh Timeout Count    :  0
Remote Notification Count:  0
Type                     :  Management</markup>

<p>Set the refreshPolicy to <code>refresh-ahead</code> for a cluster.</p>

<markup
lang="bash"

>$ cohctl set management -a refreshPolicy -v refresh-ahead -c local

Are you sure you want to set the value of attribute refreshPolicy to refresh-ahead? (y/n) y

$ cohctl get management -c local

Refresh Policy           :  refresh-ahead
Expiry Delay             :  6000
Refresh Count            :  500
Refresh Excess Count     :  143
Refresh On Query         :  false
Refresh Prediction Count :  24389
Refresh Time             :  2021-11-22T03:54:36.919Z
Refresh Timeout Count    :  0
Remote Notification Count:  0
Type                     :  Management</markup>

</div>
</div>
</div>
</doc-view>
