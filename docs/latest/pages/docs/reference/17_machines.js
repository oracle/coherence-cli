<doc-view>

<h2 id="_machines">Machines</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>There are various cluster commands that allow you display information about cluster machines.</p>

<ul class="ulist">
<li>
<p><router-link to="#get-machines" @click.native="this.scrollFix('#get-machines')"><code>cohctl get machines</code></router-link> - displays the machines for a cluster</p>

</li>
<li>
<p><router-link to="#describe-machine" @click.native="this.scrollFix('#describe-machine')"><code>cohctl describe machine</code></router-link> - shows information related to a specific machine</p>

</li>
</ul>

<h4 id="get-machines">Get Members</h4>
<div class="section">
<p>The 'get machines' command displays the machines for a cluster.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get machines [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for machines</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl get machines -c local

MACHINE                  PROCESSORS    LOAD  TOTAL MEMORY  FREE MEMORY  % FREE  OS     ARCH   VERSION
66c301108e8a/172.17.0.2           4  0.3300       8.500GB      5.858GB  68.92%  Linux  amd64  5.10.47-linuxkit</markup>

</div>

<h4 id="describe-machine">Describe Machine</h4>
<div class="section">
<p>The 'describe machine' command shows information related to a particular machine.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl describe machine machine-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for machine</pre>
</div>

<p><strong>Examples</strong></p>

<p>Describe machine 66c301108e8a/172.17.0.2.</p>

<markup
lang="bash"

>$ cohctl describe machine 66c301108e8a/172.17.0.2 -c local

Machine Name                 :  66c301108e8a/172.17.0.2
Arch                         :  amd64
Available Processors         :  4
Committed Virtual Memory Size:  6.115729408e+09
Domain                       :  java.lang
Free Physical Memory Size    :  6.284816384e+09
Free Swap Space Size         :  1.073737728e+09
Max File Descriptor Count    :  1.048576e+06
Name                         :  Linux
Node Id                      :  1
Object Name                  :  map[canonicalKeyPropertyListString:
Open File Descriptor Count   :  164
Process Cpu Load             :  0.004840661557079468
Process Cpu Time             :  1.399e+10
Sub Type                     :  OperatingSystem
System Cpu Load              :  0.03903903903903904
System Load Average          :  0.31
Total Physical Memory Size   :  9.127186432e+09
Total Swap Space Size        :  1.073737728e+09
Type                         :  Platform
Version                      :  5.10.47-linuxkit</markup>

</div>
</div>

<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/reference/15_members">Members</router-link></p>

</li>
</ul>
</div>
</div>
</doc-view>
