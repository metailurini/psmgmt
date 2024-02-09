# psmgmt
> This project provides a solution for optimizing the management of multiple system commands concurrently, addressing the limitations of the free plan on serv00.com, a server hosting platform.
> 
> By using this project, you can optimize the amount of concurrent processes and deploy multiple services simultaneously.


## Context
When deploying services on the serv00.com server using the free plan, there
is a limit on the number of concurrent processes. The current approach involves
SSH'ing into the server and running each service as a separate background
process. However, this method requires double the number of processes since
one is needed for the service and another for the SSH session. This exceeds
the process limit when multiple services are involved.

To overcome this limitation and optimize the amount of concurrent processes,
this project has been developed. It provides a more efficient approach to
managing the execution of system commands concurrently, allowing you to
run multiple services simultaneously without exceeding the process limit.
With this improvement, you can optimize the deployment of services on the
serv00.com server.


## Usage
To use this project and optimize the amount of concurrent processes, follow
the steps below:

1. Clone the project repository.
2. Ensure that Go is installed on your system.
3. Create a YAML configuration file specifying the commands to be executed
   concurrently. The file should adhere to the following format:
    ```yaml
    version: 1
    apps:
      - name: <app_name>
        command: <command_to_execute>
        args:
          - <argument_1>
          - <argument_2>
          ...
      - name: <app_name>
        command: <command_to_execute>
        args:
          - <argument_1>
          - <argument_2>
          ...
      ...
    ```
4. Build and run the project
    - Open a terminal and navigate to the project's root directory.
    - Build the project by executing the following command:

      ```shell
      go build
      ```

    - After the build is successful, run the project by executing:

      ```shell
      ./psmgmt <config_file.yml>
      ```

      Replace `<config_file.yml>` with the path to your YAML configuration file.

## Features

- [x] Optimization of concurrent execution of multiple system commands.
- [x] Capturing and streaming of command output.
- [x] Graceful shutdown with signal handling.
- [ ] Error notification when errors occur during command execution.


## Contributing
Contributions to this project are welcome. If you encounter any issues or
have suggestions for improvements, please submit an issue on the project's
GitHub repository.


## License
This project is licensed under the MIT License.
See the [LICENSE](./LICENSE) file for more information.


## Acknowledgments
This project is powered by the [serv00.com](https://www.serv00.com/) hosting
service, which provides revolutionary free hosting without ads and with
modern technologies.
