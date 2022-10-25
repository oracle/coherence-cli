<doc-view>

<h2 id="_coherence_cli_installation">Coherence CLI Installation</h2>
<div class="section">
<p>The Coherence CLI is installable on a variety of platforms and architectures including macOS, Linux and Windows.
Refer to your operating system below:</p>

<p><strong>The latest version of the CLI is 1.4.2</strong>.</p>

<ul class="ulist">
<li>
<p><router-link to="#install-macos" @click.native="this.scrollFix('#install-macos')">macOS</router-link></p>

</li>
<li>
<p><router-link to="#install-linux" @click.native="this.scrollFix('#install-linux')">Linux</router-link></p>

</li>
<li>
<p><router-link to="#install-windows" @click.native="this.scrollFix('#install-windows')">Windows</router-link></p>

</li>
</ul>
<div class="admonition note">
<p class="admonition-inline">See <a id="" title="" target="_blank" href="https://github.com/oracle/coherence-cli/releases">here</a> for all supported platforms.</p>
</div>
<p>See <router-link to="#uninstall" @click.native="this.scrollFix('#uninstall')">here</router-link> for instructions on removing <code>cohctl</code>.</p>

<div class="admonition note">
<p class="admonition-inline">If there is a platform you would like included, please submit an issue <a id="" title="" target="_blank" href="https://github.com/oracle/coherence-cli/issues/new/choose">here</a>.</p>
</div>

<h3 id="install-macos">macOS</h3>
<div class="section">
<p>Download the package installer for the latest version for either Intel or Apple Silicon (M1) directly below:</p>

<ul class="ulist">
<li>
<p>Intel - <a id="" title="" target="_blank" href="https://github.com/oracle/coherence-cli/releases/download/1.4.2/Oracle-Coherence-CLI-1.4.2-darwin-amd64.pkg">Oracle-Coherence-CLI-1.4.2-darwin-amd64.pkg</a></p>

</li>
<li>
<p>Apple Silicon - <a id="" title="" target="_blank" href="https://github.com/oracle/coherence-cli/releases/download/1.4.2/Oracle-Coherence-CLI-1.4.2-darwin-arm64.pkg">Oracle-Coherence-CLI-1.4.2-darwin-arm64.pkg</a></p>

</li>
</ul>
<div class="admonition note">
<p class="admonition-inline">When you install the pkg it will place the <code>cohctl</code> executable in the <code>/usr/local/bin</code> directory. You can move it out of this directory if you wish after installation.</p>
</div>
</div>

<h3 id="install-linux">Linux</h3>
<div class="section">
<p>Install the latest release using curl on Linux:</p>

<markup
lang="bash"

>curl -Lo cohctl "https://github.com/oracle/coherence-cli/releases/download/1.4.2/cohctl-1.4.2-linux-amd64"
chmod u+x ./cohctl</markup>

<p>You can move the executable to your preferred location afterwards.</p>

<div class="admonition note">
<p class="admonition-inline">Change the <code>amd64</code> to <code>arm64</code> for ARM based processor, or to <code>386</code> for i386 processors.</p>
</div>
<p>To install a specific release, change the version number in the above command.</p>

</div>

<h3 id="install-windows">Windows</h3>
<div class="section">
<p>Install the latest version using curl on Windows and copy to a directory in your PATH:</p>

<markup
lang="bash"

>curl -Lo cohctl.exe "https://github.com/oracle/coherence-cli/releases/download/1.4.2/cohctl-1.4.2-windows-amd64.exe"</markup>

<div class="admonition note">
<p class="admonition-inline">Change the <code>amd64</code> to <code>arm</code> for ARM based processor.</p>
</div>
</div>

<h3 id="uninstall">Removing the CLI</h3>
<div class="section">
<p>To uninstall <code>cohctl</code>, for Mac, issue the following:</p>

<markup
lang="command"

>sudo rm /usr/local/bin/cohctl</markup>

<p>For all other platforms, remove the <code>cohctl</code> executable from where you copied it to.</p>

<p>If you also wish to uninstall the configuration directory, <code>.cohctl</code>, where <code>cohctl</code> stores its configuration,
you will find this off the users home directory. See <router-link to="/docs/config/10_changing_config_locations">here</router-link> for more information.</p>

</div>

<h3 id="_next_steps">Next Steps</h3>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/about/03_quickstart">Run the Quick Start</router-link></p>

</li>
</ul>
</div>
</div>
</doc-view>
