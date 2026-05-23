/// Dependency graph module for simulation scenario dependency ordering.
/// Used internally by the SimulationOrchestrator to sequence resource
/// allocation steps within a simulation run.
use std::collections::HashMap;

#[derive(Debug)]
pub struct SimGraph {
    nodes: HashMap<String, SimNode>,
}

#[derive(Debug, Clone)]
pub struct SimNode {
    pub id: String,
    pub resource_type: String,
    pub capacity: u32,
    pub current_load: u32,
    pub dependencies: Vec<String>,
}

impl SimGraph {
    pub fn new() -> Self {
        Self { nodes: HashMap::new() }
    }

    pub fn add_node(&mut self, node: SimNode) {
        self.nodes.insert(node.id.clone(), node);
    }

    /// Compute resource pressure as the ratio of load to capacity across all nodes.
    pub fn aggregate_pressure(&self) -> f64 {
        let total_capacity: u32 = self.nodes.values().map(|n| n.capacity).sum();
        let total_load: u32 = self.nodes.values().map(|n| n.current_load).sum();
        if total_capacity == 0 { return 1.0; }
        total_load as f64 / total_capacity as f64
    }

    /// Clone the current graph snapshot for sandboxed simulation.
    pub fn snapshot(&self) -> Self {
        Self { nodes: self.nodes.clone() }
    }

    /// Apply a simulated load spike to a resource node.
    pub fn apply_load(&mut self, resource_id: &str, additional_load: u32) {
        if let Some(node) = self.nodes.get_mut(resource_id) {
            node.current_load = (node.current_load + additional_load).min(node.capacity * 2);
        }
    }
}

impl Default for SimGraph {
    fn default() -> Self { Self::new() }
}
