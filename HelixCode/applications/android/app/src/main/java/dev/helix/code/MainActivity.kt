package dev.helix.code

import android.os.Bundle
import android.widget.Button
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import org.json.JSONArray
import org.json.JSONObject

class MainActivity : AppCompatActivity() {

    private lateinit var connectionStatusText: TextView
    private lateinit var userText: TextView
    private lateinit var tasksRecyclerView: RecyclerView
    private lateinit var connectButton: Button

    private val tasks = mutableListOf<Task>()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        // Initialize views
        connectionStatusText = findViewById(R.id.connectionStatusText)
        userText = findViewById(R.id.userText)
        tasksRecyclerView = findViewById(R.id.tasksRecyclerView)
        connectButton = findViewById(R.id.connectButton)

        // Setup RecyclerView
        tasksRecyclerView.layoutManager = LinearLayoutManager(this)
        tasksRecyclerView.adapter = TaskAdapter(tasks) { task ->
            showTaskDetails(task)
        }

        // Setup button click
        connectButton.setOnClickListener {
            toggleConnection()
        }

        // Initialize mobile core
        MobileCore.shared.initialize()

        // Update UI
        updateUI()
    }

    private fun toggleConnection() {
        if (MobileCore.shared.isConnected) {
            MobileCore.shared.disconnect()
        } else {
            // For demo purposes, connect with test credentials
            val connected = MobileCore.shared.connect(
                "http://localhost:8080",
                "testuser",
                "testpass"
            )
            if (!connected) {
                showAlert("Connection Failed", "Could not connect to server")
            }
        }
        updateUI()
    }

    private fun updateUI() {
        // Update connection status
        val isConnected = MobileCore.shared.isConnected
        connectionStatusText.text = if (isConnected) "Connected" else "Disconnected"
        connectionStatusText.setTextColor(
            if (isConnected) android.graphics.Color.GREEN else android.graphics.Color.RED
        )

        // Update user
        userText.text = "User: ${MobileCore.shared.currentUser}"

        // Update tasks
        loadTasks()

        // Update button
        connectButton.text = if (isConnected) "Disconnect" else "Connect"
    }

    private fun loadTasks() {
        val tasksJson = MobileCore.shared.tasks
        tasks.clear()

        try {
            val jsonArray = JSONArray(tasksJson)
            for (i in 0 until jsonArray.length()) {
                val taskObj = jsonArray.getJSONObject(i)
                val task = Task(
                    id = taskObj.getString("id"),
                    name = taskObj.getString("name"),
                    description = taskObj.getString("description"),
                    status = taskObj.getString("status"),
                    progress = taskObj.getInt("progress")
                )
                tasks.add(task)
            }
        } catch (e: Exception) {
            e.printStackTrace()
        }

        tasksRecyclerView.adapter?.notifyDataSetChanged()
    }

    private fun showTaskDetails(task: Task) {
        val dialog = android.app.AlertDialog.Builder(this)
            .setTitle(task.name)
            .setMessage(task.description)
            .setPositiveButton("OK", null)
            .create()
        dialog.show()
    }

    private fun showAlert(title: String, message: String) {
        val dialog = android.app.AlertDialog.Builder(this)
            .setTitle(title)
            .setMessage(message)
            .setPositiveButton("OK", null)
            .create()
        dialog.show()
    }

    override fun onDestroy() {
        super.onDestroy()
        MobileCore.shared.close()
    }
}

data class Task(
    val id: String,
    val name: String,
    val description: String,
    val status: String,
    val progress: Int
)