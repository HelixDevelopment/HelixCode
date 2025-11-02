//
//  AppDelegate.swift
//  HelixCode
//
//  Created by HelixCode on 2025-11-02.
//

import UIKit

@main
class AppDelegate: UIResponder, UIApplicationDelegate {

    var window: UIWindow?

    func application(_ application: UIApplication, didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?) -> Bool {
        // Initialize the mobile core
        MobileCore.shared.initialize()

        return true
    }

    func applicationWillTerminate(_ application: UIApplication) {
        // Cleanup
        MobileCore.shared.close()
    }
}