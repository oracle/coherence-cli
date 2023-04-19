<doc-view>

<h2 id="_coherence_cli_installation">Coherence CLI Installation</h2>
<div class="section">
<p>The Coherence CLI is installable on a variety of platforms and architectures including macOS, Linux and Windows.
Refer to your operating system below:</p>

<p><strong>The latest version of the CLI is 1.5.0</strong>.</p>

<ul class="ulist">
<li>
<p><router-link to="#install-macos-linux" @click.native="this.scrollFix('#install-macos-linux')">Install on macOS or Linux</router-link></p>

</li>
<li>
<p><router-link to="#install-windows" @click.native="this.scrollFix('#install-windows')">Install on Windows</router-link></p>

</li>
<li>
<p><router-link to="#uninstall" @click.native="this.scrollFix('#uninstall')">Removing the CLI</router-link></p>

</li>
</ul>
<p>See <a id="" title="" target="_blank" href="https://github.com/oracle/coherence-cli/releases">here</a> for all supported platforms.</p>

<div class="admonition note">
<p class="admonition-inline">If there is a platform you would like included, please submit an issue <a id="" title="" target="_blank" href="https://github.com/oracle/coherence-cli/issues/new/choose">here</a>.</p>
</div>

<h3 id="install-macos-linux">macOS or Linux</h3>
<div class="section">
<p>For macOS or Linux platforms, use the following to install the latest version of the CLI:</p>

<markup
lang="bash"

>curl -sL https://raw.githubusercontent.com/oracle/coherence-cli/main/scripts/install.sh | bash</markup>

<div class="admonition note">
<p class="admonition-inline">When you install the CLI, administrative privileges are required as the <code>cohctl</code> executable is moved to the <code>/usr/local/bin</code> directory. You can move it out of this directory if you wish after installation.</p>
</div>
</div>

<h3 id="install-windows">Windows</h3>
<div class="section">
<p>Install the latest version using the <code>curl</code> command below and copy to a directory in your <code>PATH</code>:
Alternativley you can download the relevant exe from <a id="" title="" target="_blank" href="https://github.com/oracle/coherence-cli/releases">here</a>.</p>

<markup
lang="bash"

>curl -Lo cohctl.exe "https://github.com/oracle/coherence-cli/releases/download/1.5.0/cohctl-1.5.0-windows-amd64.exe"</markup>

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

<p>If you also wish to uninstall the hidden configuration directory, <code>.cohctl</code>, where <code>cohctl</code> stores its configuration,
you will find this off the users home directory. See <router-link to="/docs/config/10_changing_config_locations">here</router-link> for more information.</p>

<p>For example on macOS or Linux the directory is <code>$HOME/.cohctl</code>, for Windows it will be under <code>%HOME%\.cohctl</code>.</p>

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
