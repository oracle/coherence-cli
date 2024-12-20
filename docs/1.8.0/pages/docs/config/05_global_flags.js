<doc-view>

<h2 id="_global_flags">Global Flags</h2>
<div class="section">
<p>The Coherence CLI provides a number of global flags that are available in all the commands. These flags are described below.</p>

<markup
lang="bash"

>Flags:
  -b, --bytes               show sizes in bytes
      --config string       config file (default is $HOME/.cohctl/cohctl.yaml)
      --config-dir string   config directory (default is $HOME/.cohctl)
  -c, --connection string   cluster connection name (not required if context is set)
  -d, --delay int32         delay for watching in seconds (default 5)
      --desc                indicates descending sort for tables, default is ascending
  -g, --gb                  show sizes in gigabytes (default is bytes)
  -h, --help                help for cohctl
  -k, --kb                  show sizes in kilobytes (default is bytes)
      --limit               limit table output to screen size
  -m, --mb                  show sizes in megabytes (default is bytes)
  -o, --output string       output format: table, wide, json or jsonpath="..." (default "table")
      --sort string         specify a sorting column name or number for tables
  -i, --stdin               read password from stdin
      --tb                  show sizes in terabytes (default is bytes)
  -U, --username string     basic auth username if authentication is required
  -w, --watch               watch output (only available for get commands)
  -W, --watch-clear         watch output with clear</markup>


<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/config/10_changing_config_locations">Changing Config Locations</router-link></p>

</li>
<li>
<p><router-link to="/docs/examples/10_jsonpath">Using JsonPath</router-link></p>

</li>
<li>
<p><router-link to="/docs/config/08_sorting_table_output">Sorting Table Output</router-link></p>

</li>
</ul>
</div>
</div>
</doc-view>
