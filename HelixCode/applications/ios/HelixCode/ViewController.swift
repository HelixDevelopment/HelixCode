//
//  ViewController.swift
//  HelixCode
//
//  Created by HelixCode on 2025-11-02.
//

import UIKit

class ViewController: UIViewController {

    @IBOutlet weak var connectionStatusLabel: UILabel!
    @IBOutlet weak var userLabel: UILabel!
    @IBOutlet weak var tasksTableView: UITableView!
    @IBOutlet weak var connectButton: UIButton!

    private var tasks: [[String: Any]] = []

    override func viewDidLoad() {
        super.viewDidLoad()

        // Setup UI
        setupUI()

        // Load initial data
        updateUI()
    }

    private func setupUI() {
        title = "HelixCode"

        tasksTableView.delegate = self
        tasksTableView.dataSource = self
        tasksTableView.register(UITableViewCell.self, forCellReuseIdentifier: "TaskCell")

        connectButton.addTarget(self, action: #selector(connectTapped), for: .touchUpInside)
    }

    private func updateUI() {
        // Update connection status
        let isConnected = MobileCore.shared.isConnected()
        connectionStatusLabel.text = isConnected ? "Connected" : "Disconnected"
        connectionStatusLabel.textColor = isConnected ? .green : .red

        // Update user
        userLabel.text = "User: \(MobileCore.shared.getCurrentUser())"

        // Update tasks
        tasks = MobileCore.shared.getTasks() ?? []
        tasksTableView.reloadData()

        // Update button
        connectButton.setTitle(isConnected ? "Disconnect" : "Connect", for: .normal)
    }

    @objc private func connectTapped() {
        if MobileCore.shared.isConnected() {
            MobileCore.shared.disconnect()
        } else {
            // For demo purposes, connect with test credentials
            let connected = MobileCore.shared.connect(serverURL: "http://localhost:8080",
                                                    username: "testuser",
                                                    password: "testpass")
            if !connected {
                showAlert(title: "Connection Failed", message: "Could not connect to server")
            }
        }
        updateUI()
    }

    private func showAlert(title: String, message: String) {
        let alert = UIAlertController(title: title, message: message, preferredStyle: .alert)
        alert.addAction(UIAlertAction(title: "OK", style: .default))
        present(alert, animated: true)
    }
}

extension ViewController: UITableViewDelegate, UITableViewDataSource {
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        return tasks.count
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(withIdentifier: "TaskCell", for: indexPath)

        if let task = tasks[indexPath.row] as? [String: Any],
           let name = task["name"] as? String,
           let status = task["status"] as? String {
            cell.textLabel?.text = "\(name) - \(status)"
        }

        return cell
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)

        if let task = tasks[indexPath.row] as? [String: Any],
           let name = task["name"] as? String,
           let description = task["description"] as? String {
            showAlert(title: name, message: description)
        }
    }
}