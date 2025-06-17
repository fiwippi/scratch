use std::ops::{Add, AddAssign, Div, DivAssign, Mul, MulAssign, Neg, Sub, SubAssign};

#[derive(Debug, Copy, Clone, PartialEq)]
pub struct Vec2 {
    pub x: f64,
    pub y: f64,
}

// Methods

impl Vec2 {
    pub fn new(x: f64, y: f64) -> Self {
        Self { x, y }
    }

    pub fn zero() -> Self {
        Self::new(0.0, 0.0)
    }

    pub fn mag(self) -> f64 {
        f64::sqrt(self.x.powi(2) + self.y.powi(2))
    }

    pub fn norm(self) -> Self {
        if self.mag() < 1e-3 {
            return Self::zero();
        }

        Self {
            x: self.x / self.mag(),
            y: self.y / self.mag()
        }
    }
}

// Operator Traits

impl Neg for Vec2 {
    type Output = Self;

    fn neg(self) -> Self::Output {
        Self::Output {
            x: -self.x,
            y: -self.y
        }
    }
}

impl Add for Vec2 {
    type Output = Self;

    fn add(self, rhs: Self) -> Self::Output {
        Self {
            x: self.x + rhs.x,
            y: self.y + rhs.y
        }
    }
}

impl AddAssign<Vec2> for Vec2 {
    fn add_assign(&mut self, rhs: Self) {
        self.x += rhs.x;
        self.y += rhs.y;
    }
}

impl Sub for Vec2 {
    type Output = Self;

    fn sub(self, rhs: Self) -> Self::Output {
        Self {
            x: self.x - rhs.x,
            y: self.y - rhs.y
        }
    }
}

impl SubAssign<Vec2> for Vec2 {
    fn sub_assign(&mut self, rhs: Self) {
        self.x -= rhs.x;
        self.y -= rhs.y;
    }
}

impl Mul<f64> for Vec2 {
    type Output = Self;

    fn mul(self, rhs: f64) -> Self::Output {
        Self {
            x: self.x * rhs,
            y: self.y * rhs
        }
    }
}

impl MulAssign<f64> for Vec2 {
    fn mul_assign(&mut self, rhs: f64) {
        self.x *= rhs;
        self.y *= rhs;
    }
}

impl Div<f64> for Vec2 {
    type Output = Self;

    fn div(self, rhs: f64) -> Self::Output {
        Self {
            x: self.x / rhs,
            y: self.y / rhs
        }
    }
}

impl DivAssign<f64> for Vec2 {
    fn div_assign(&mut self, rhs: f64) {
        self.x /= rhs;
        self.y /= rhs;
    }
}

// Tests

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn add() {
        // Positive
        let v1 = Vec2::new(1.0, 2.0);
        let v2 = Vec2::new(2.0, 1.0);
        assert_eq!(Vec2::new(3.0, 3.0), v1 + v2);

        // Negative
        let v1 = Vec2::new(-1.0, -2.0);
        let v2 = Vec2::new(-2.0, -1.0);
        assert_eq!(Vec2::new(-3.0, -3.0), v1 + v2);
    }

    #[test]
    fn mag() {
        let v1 = Vec2::new(-5.0, -5.0);
        let v2 = Vec2::new(5.0, 5.0);
        assert_eq!(f64::sqrt(200.0), (v1 - v2).mag());
        assert_eq!(f64::sqrt(200.0), (v2 - v1).mag());
    }
}

