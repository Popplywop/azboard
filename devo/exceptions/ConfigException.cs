namespace devo.exceptions;

/// <summary>
/// Thrown when the config file is missing, malformed, or fails validation
/// </summary>
public sealed class ConfigException : Exception
{
    public ConfigException(string message) : base(message) { }
    public ConfigException(string message, Exception inner) : base(message, inner) { }
}