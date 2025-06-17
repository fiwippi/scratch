use wasm_bindgen::prelude::*;
use web_sys::CanvasRenderingContext2d;

use super::boid::Boid;

const DRAW_INTERVAL: f64 = 1.0 / 60.0 * 1000.0; // Max of 60fps, i.e. every 16 milliseconds

#[wasm_bindgen]
pub struct Flock {
    area_width: f64,
    area_height: f64,
    boids: Vec<Boid>,
    since_last_update: f64,
}

#[wasm_bindgen]
impl Flock {
    #[wasm_bindgen(constructor)]
    pub fn new(area_width: f64, area_height: f64, size: usize) -> Self {
        let mut f = Flock { area_width, area_height, boids: Vec::new(), since_last_update: 0.0 };
        f.add_boids(size);
        f
    }

    pub fn add_boids(&mut self, count: usize) {
        for _ in 0..count {
            let boid_x = rand::random::<f64>() * self.area_width;
            let boid_y = rand::random::<f64>() * self.area_height;
            self.boids.push(Boid::new(boid_x, boid_y));
        }
    }

    pub fn change_size(&mut self, count: usize) -> Result<(), JsValue> {
        let delta = (count - self.boids.len()) as isize;
        match delta > 0 {
            true => {
                // Growing
                self.add_boids(delta as usize);
            }
            false => {
                // Shrinking - We don't expect to add the zeroed boid
                self.boids.resize(count, Boid::new(0.0, 0.0));
            }
        }

        Ok(())
    }

    pub fn draw(&mut self, ctx: &CanvasRenderingContext2d, tstamp: f64, draw_velocity: bool, draw_neighbourhood: bool, neighbourhood_area: f64) -> Result<(), JsValue> {
        // Clear the canvas before drawing
        ctx.clear_rect(0.0, 0.0, self.area_width, self.area_height);

        // Step the boids
        let delta: f64 = tstamp - self.since_last_update;
        if delta >= DRAW_INTERVAL {
            self.since_last_update = tstamp;

            let cloned = self.boids.clone();

            for (i, b) in self.boids.iter_mut().enumerate() {
                // Get all boids which are not the boid we are trying to move
                let mut bs: Vec<Boid> = Vec::new();
                for (j, &other) in cloned.iter().enumerate() {
                    if i != j {
                        bs.push(other);
                    }
                }

                // Change the velocity of the boid
                b.step(&bs[..], delta/1000.0, neighbourhood_area, self.area_width, self.area_height);
            }
        }

        // Draw all boids:
        // 1. First draw neighbourhood
        // 2. Then velocity
        // 3. Then body

        // Step 1
        if draw_neighbourhood {
            for b in self.boids.iter() {
                b.draw_neighbourhood(ctx)?;
            }
        }
        // Step 2
        if draw_velocity {
            for b in self.boids.iter() {
                b.draw_velocity(ctx)?;
            }
        }

        // Step 3
        for b in self.boids.iter() {
            b.draw_body(ctx)?;
        }

        Ok(())
    }
}

