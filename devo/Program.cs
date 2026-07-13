using System.Reflection;

using ConsoleForge.Core;
using ConsoleForge.Styling;

using devo.config;
using devo.exceptions;
using devo.services;
using devo.ui;

string version = Assembly.GetExecutingAssembly()
    .GetCustomAttribute<AssemblyInformationalVersionAttribute>()?.InformationalVersion ?? "dev";

int jumpToPrID = 0;
for (int i = 0; i < args.Length; i++)
{
    switch (args[i])
    {
        case "--version" or "-v":
            await Console.Out.WriteLineAsync($"devo {version}");
            return 0;
        case "--pr":
            if (i + 1 >= args.Length || !int.TryParse(args[i+1], out jumpToPrID))
            {
                await Console.Error.WriteLineAsync("Error: --pr requires an integer PR ID");
                return 1;
            }
            i++;
            break;
    }
}

IAdoClient client;
Config config;

try
{
    config = await Config.LoadAsync();
}
catch (ConfigException e)
{
    await Console.Error.WriteLineAsync($"Error: {e.Message}\n");
    await Console.Error.WriteLineAsync($"Create a config file at {Config.ConfigFilePath()}:");
    await Console.Error.WriteLineAsync("""
                  {
                    "auth_method": "pat",
                    "org_url": "https://dev.azure.com/yourorg",
                    "project": "YourProject",
                    "pat": "your-personal-access-token"
                  }
              """);
    return 1;
}

var http = new HttpClient { Timeout = TimeSpan.FromSeconds(30) };
client = new CachedAdoClient(new AdoClient(config, http));

try
{
    await App.Run(new AppModel(client, config, jumpToPrID, version), theme: Theme.Dracula);
    return 0;
}
catch (Exception ex)
{
    await Console.Error.WriteLineAsync($"Error runnint TUI: {ex}");
    return 1;
}