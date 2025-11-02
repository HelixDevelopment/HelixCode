package dev.helix.code

import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ProgressBar
import android.widget.TextView
import androidx.recyclerview.widget.RecyclerView

class TaskAdapter(
    private val tasks: List<Task>,
    private val onTaskClick: (Task) -> Unit
) : RecyclerView.Adapter<TaskAdapter.TaskViewHolder>() {

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): TaskViewHolder {
        val view = LayoutInflater.from(parent.context)
            .inflate(R.layout.item_task, parent, false)
        return TaskViewHolder(view)
    }

    override fun onBindViewHolder(holder: TaskViewHolder, position: Int) {
        val task = tasks[position]
        holder.bind(task, onTaskClick)
    }

    override fun getItemCount(): Int = tasks.size

    class TaskViewHolder(itemView: View) : RecyclerView.ViewHolder(itemView) {
        private val nameText: TextView = itemView.findViewById(R.id.taskNameText)
        private val statusText: TextView = itemView.findViewById(R.id.taskStatusText)
        private val progressBar: ProgressBar = itemView.findViewById(R.id.taskProgressBar)

        fun bind(task: Task, onTaskClick: (Task) -> Unit) {
            nameText.text = task.name
            statusText.text = task.status
            progressBar.progress = task.progress

            itemView.setOnClickListener {
                onTaskClick(task)
            }
        }
    }
}