using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Linq;
using System.Text;
using Newtonsoft.Json;
using System.Security.Cryptography;

namespace HackLang_Tools
{
    class Program
    {
        static readonly string Suffix = "hhtools";
        static readonly string ConfigFileName = "." + Suffix;

        static Uri ExecPath;
        static Uri ProjectPath;
        static Uri ConfigPath;
        static Config CurrentConfig;
        static string ContainerName;
        static bool JSONOutput;

        static void Main(string[] args)
        {
            JSONOutput = Environment.GetCommandLineArgs.Contains("--json")
            ExecPath = new Uri(System.IO.Path.GetDirectoryName(
                                                    System.IO.Path.GetFullPath(Environment.GetCommandLineArgs().First())));
            ConfigPath = FindConfigFile(ExecPath.LocalPath, ConfigFileName);
            ProjectPath = new Uri(System.IO.Path.GetDirectoryName(ConfigPath.LocalPath));
            CurrentConfig = ParseConfig(ConfigPath);
            ContainerName = Suffix + "_" + Hash(ProjectPath.AbsolutePath);

            ExecInDockerContainer();
        }
        static void ExecInDockerContainer()
        {
            Process Docker = new Process
            {
                StartInfo = new ProcessStartInfo
                {
                    FileName = FindProgram("docker.exe"),
                    Arguments = String.Join(" ", new List<string>() {"exec", ContainerName, "/bin/sh", "-c",
                        "\"cd " + TranslatePathToUNIX(ExecPath).AbsolutePath + "; " + Process.GetCurrentProcess().ProcessName + 
                        " " + String.Join(" ", Environment.GetCommandLineArgs().Skip(1))
                        + "\""}),
                    UseShellExecute = false,
                    RedirectStandardError = true,
                    RedirectStandardOutput = true,
                }
            };

            try
            {
                Docker.OutputDataReceived += ProcessDockerOutput;
                Docker.ErrorDataReceived += ProcessDockerOutput;

                Docker.Start();

                Docker.BeginErrorReadLine();
                Docker.BeginOutputReadLine();
                Docker.WaitForExit();
            }
            catch (Exception e)
            {
                Console.Error.WriteLine(e.Message);
                Environment.Exit(1);
            }
        }
        static void ProcessDockerOutput(object sender, DataReceivedEventArgs e)
        {
            if (e.Data == null) return;
            if (e.Data.EndsWith("is not running"))
            {
                Console.Error.WriteLine("Docker container is not running. Going to start it");
                StartDockerContainer();
                ExecInDockerContainer();
            }
            else if (e.Data.Contains("No such container"))
            {
                CreateDockerContainer();
                ExecInDockerContainer();
            }
            else
            {
                Console.WriteLine(e.Data);
            }
        }
        static void StartDockerContainer()
        {
            Process Docker = new Process
            {
                StartInfo = new ProcessStartInfo
                {
                    FileName = FindProgram("docker.exe"),
                    Arguments = String.Join(" ", new List<string>() {"start", ContainerName}),
                    UseShellExecute = false
                }
            };

            try
            {
                Docker.Start();
            }
            catch (Exception e)
            {
                Console.Error.WriteLine(e.Message);
                Environment.Exit(1);
            }
            System.Threading.Thread.Sleep(300);
        }
        static void CreateDockerContainer()
        {
            string ProjectUnixPath = TranslatePathToUNIX(ProjectPath).AbsolutePath
            Process Docker = new Process
            {
                StartInfo = new ProcessStartInfo
                {
                    FileName = FindProgram("docker.exe"),
                    Arguments = String.Join(" ", new List<string>() {"run", "-d", "-t", "--name=" + ContainerName,
                        "-v=" +ProjectUnixPath+":"+ProjectUnixPath, "-w=" + ProjectUnixPath, CurrentConfig.image }),
                    UseShellExecute = false
                }
            };

            try
            {
                Docker.Start();
            }
            catch (Exception e)
            {
                Console.Error.WriteLine(e.Message);
                Environment.Exit(1);
            }
            System.Threading.Thread.Sleep(500);
        }
        static Config ParseConfig(Uri configPath)
        {
            try
            {
                return JsonConvert.DeserializeObject<Config>(System.IO.File.ReadAllText(configPath.LocalPath));
            }
            catch
            {
                Console.Error.WriteLine("Invalid config file at " + configPath.LocalPath + " . Make sure that the file " +
                                        "contains valid JSON configrution.");
                Environment.Exit(1);
            }
            return new Config();
        }
        static Uri FindConfigFile(string findPath, string fileName)
        {
            if (System.IO.File.Exists(System.IO.Path.Combine(findPath, fileName)))
            {
                return new Uri(System.IO.Path.Combine(findPath, fileName));
            }
            else
            {
                if (findPath == System.IO.Path.GetPathRoot(findPath))
                {
                    Console.Error.WriteLine("No " + ConfigFileName + " config file was found in this or any parent directory.");
                    Environment.Exit(1);
                }

                return FindConfigFile(System.IO.Directory.GetParent(findPath).FullName, fileName);
            }
        }
        static Uri TranslatePathToUNIX(Uri localPath)
        {
            UriBuilder uriConstruct = new UriBuilder();
                       uriConstruct.Scheme = Uri.UriSchemeFile;

            string drive = System.IO.Path.GetPathRoot(localPath.AbsolutePath);

                       uriConstruct.Path = System.IO.Path.AltDirectorySeparatorChar +
                                           drive.First().ToString().ToLower() + System.IO.Path.AltDirectorySeparatorChar +
                                           localPath.AbsolutePath.Substring(drive.Length);

            return uriConstruct.Uri;
        }
        static Uri TranslatePathToWindows(Uri remotePath)
        {
            string drive = remotePath.AbsolutePath.Substring(1, 1).ToUpper() + System.IO.Path.VolumeSeparatorChar + 
                           System.IO.Path.AltDirectorySeparatorChar;

            return new Uri(drive + remotePath.AbsolutePath.Substring(drive.Length));
        }
        static string FindProgram(string name)
        {
            Process Where = new Process
            {
                StartInfo = new ProcessStartInfo
                {
                    FileName = @"C:\Windows\System32\where.exe",
                    Arguments = name,
                    UseShellExecute = false,
                    RedirectStandardOutput = true,
                    CreateNoWindow = true
                }
            };

            try
            {
                Where.Start();
                while (!Where.StandardOutput.EndOfStream)
                {
                    return (string)Where.StandardOutput.ReadLine();
                }
                throw new Exception(name + " was not found in your PATH");
            }
            catch (Exception e)
            {
                Console.Error.WriteLine(e.Message);
                Environment.Exit(1);
                return "";
            }
        }
        static string Hash(string input)
        {
            return string.Join("", (new SHA1Managed()).ComputeHash(Encoding.UTF8.GetBytes(input)).Select(b => b.ToString("x2")).ToArray());
        }

        public struct Config
        {
            public string provider;
            public string image;
        }
    }
}
