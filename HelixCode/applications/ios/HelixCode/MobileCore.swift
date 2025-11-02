//
//  MobileCore.swift
//  HelixCode
//
//  Created by HelixCode on 2025-11-02.
//

import Foundation

// MobileCore provides access to the shared Go mobile core
class MobileCore {
    static let shared = MobileCore()

    private var core: HelixCoreMobileCore?

    private init() {
        // Initialize the Go mobile core
        core = HelixCoreNewMobileCore()
    }

    func initialize() {
        guard let core = core else { return }
        do {
            try core.initialize()
        } catch {
            print("Failed to initialize mobile core: \(error)")
        }
    }

    func connect(serverURL: String, username: String, password: String) -> Bool {
        guard let core = core else { return false }
        do {
            try core.connect(serverURL, username: username, password: password)
            return true
        } catch {
            print("Failed to connect: \(error)")
            return false
        }
    }

    func disconnect() {
        guard let core = core else { return }
        do {
            try core.disconnect()
        } catch {
            print("Failed to disconnect: \(error)")
        }
    }

    func isConnected() -> Bool {
        guard let core = core else { return false }
        return core.isConnected()
    }

    func getCurrentUser() -> String {
        guard let core = core else { return "" }
        return core.getCurrentUser()
    }

    func getDashboardData() -> [String: Any]? {
        guard let core = core else { return nil }
        let jsonString = core.getDashboardData()
        return parseJSON(jsonString)
    }

    func getTasks() -> [[String: Any]]? {
        guard let core = core else { return nil }
        let jsonString = core.getTasks()
        guard let data = parseJSON(jsonString) as? [String: Any],
              let tasks = data["tasks"] as? [[String: Any]] else {
            return nil
        }
        return tasks
    }

    func getWorkers() -> [[String: Any]]? {
        guard let core = core else { return nil }
        let jsonString = core.getWorkers()
        guard let data = parseJSON(jsonString) as? [String: Any],
              let workers = data["workers"] as? [[String: Any]] else {
            return nil
        }
        return workers
    }

    func createTask(name: String, description: String) -> [String: Any]? {
        guard let core = core else { return nil }
        let jsonString = core.createTask(name, description: description)
        return parseJSON(jsonString)
    }

    func sendNotification(title: String, message: String, type: String) -> Bool {
        guard let core = core else { return false }
        let jsonString = core.sendNotification(title, message: message, type: type)
        guard let data = parseJSON(jsonString) as? [String: Any],
              let success = data["success"] as? Bool else {
            return false
        }
        return success
    }

    func getTheme() -> [String: Any]? {
        guard let core = core else { return nil }
        let jsonString = core.getTheme()
        return parseJSON(jsonString)
    }

    func setTheme(_ themeName: String) -> Bool {
        guard let core = core else { return false }
        return core.setTheme(themeName)
    }

    func getAvailableThemes() -> [String]? {
        guard let core = core else { return nil }
        let jsonString = core.getAvailableThemes()
        guard let data = parseJSON(jsonString) as? [String: Any],
              let themes = data["themes"] as? [String] else {
            return nil
        }
        return themes
    }

    func close() {
        guard let core = core else { return }
        do {
            try core.close()
        } catch {
            print("Failed to close mobile core: \(error)")
        }
    }

    private func parseJSON(_ jsonString: String) -> Any? {
        guard let data = jsonString.data(using: .utf8) else { return nil }
        do {
            return try JSONSerialization.jsonObject(with: data, options: [])
        } catch {
            print("Failed to parse JSON: \(error)")
            return nil
        }
    }
}