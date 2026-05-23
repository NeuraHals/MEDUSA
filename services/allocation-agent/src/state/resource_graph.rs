/// ResourceGraph provides Redis Graph Cypher query helpers for resource state
/// traversal up to the configured max depth.
/// In production, this uses redis::cmd("GRAPH.QUERY") against the RedisGraph module.
pub struct ResourceGraph;

impl ResourceGraph {
    /// Returns the Cypher query string for available resources in a hospital.
    pub fn available_resources_query(hospital_id: &str, max_depth: u32) -> String {
        format!(
            "MATCH (r:Resource {{hospital_id: '{}', status: 'AVAILABLE'}}) \
             WHERE r.capacity_available > 0 \
             RETURN r LIMIT {}",
            hospital_id,
            max_depth * 10
        )
    }

    /// Returns a query to check if a resource is still in AVAILABLE state
    /// (used for optimistic concurrency pre-lock check).
    pub fn resource_status_query(resource_id: &str) -> String {
        format!(
            "MATCH (r:Resource {{id: '{}'}}) RETURN r.status, r.version",
            resource_id
        )
    }
}
