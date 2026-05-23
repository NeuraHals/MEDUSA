/// Bayesian confidence updater.
///
/// Uses the formula:
///   P(crisis | evidence) = P(evidence | crisis) * P(crisis) / P(evidence)
///
/// Implemented as iterative Bayesian updates where each new contributing
/// signal updates the posterior probability.
pub struct BayesianUpdater {
    /// Prior probability of a genuine crisis for this crisis type (0.0–1.0)
    prior: f64,
}

impl BayesianUpdater {
    pub fn new(prior: f64) -> Self {
        Self { prior }
    }

    /// Update the posterior with a new piece of evidence.
    /// `likelihood` = P(signal_observed | crisis_real) — source reliability weight.
    /// `false_positive_rate` = P(signal_observed | crisis_false).
    pub fn update(&self, posterior: f64, likelihood: f64, false_positive_rate: f64) -> f64 {
        let p_evidence = (likelihood * posterior)
            + (false_positive_rate * (1.0 - posterior));

        if p_evidence == 0.0 {
            return posterior;
        }

        (likelihood * posterior) / p_evidence
    }

    /// Run iterative Bayesian updates over a slice of signal reliabilities.
    /// Returns the final posterior after all evidence is incorporated.
    pub fn run(&self, signal_reliabilities: &[f64]) -> f64 {
        let mut posterior = self.prior;
        for &reliability in signal_reliabilities {
            // False positive rate assumed as (1 - reliability) / 2
            let fpr = (1.0 - reliability) / 2.0;
            posterior = self.update(posterior, reliability, fpr);
            posterior = posterior.clamp(0.0, 1.0);
        }
        posterior
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_bayesian_update_increases_with_high_reliability() {
        let updater = BayesianUpdater::new(0.3);
        let result = updater.run(&[0.90, 0.92, 0.88]);
        assert!(result > 0.3, "posterior should increase with reliable signals");
        assert!(result <= 1.0, "posterior must not exceed 1.0");
    }

    #[test]
    fn test_bayesian_update_decreases_with_low_reliability() {
        let updater = BayesianUpdater::new(0.9);
        let result = updater.run(&[0.10, 0.15]);
        assert!(result < 0.9, "posterior should decrease with unreliable signals");
    }
}
