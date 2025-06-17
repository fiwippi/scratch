use std::sync::Mutex;
use wasm_bindgen::prelude::*;
use web_sys::CanvasRenderingContext2d;

use crate::vectors::Vec2;

#[wasm_bindgen]
#[derive(Debug, Copy, Clone, PartialEq)]
pub struct Boid {
    pos: Vec2,
    vel: Vec2,

}

static NEIGHBOURHOOD_AREA: Mutex<f64> = Mutex::new(0.0);

impl Boid {
    pub fn new(x: f64, y: f64) -> Self {
        Self {
            pos: Vec2::new(x, y),
            vel: Vec2::new(rand::random::<f64>(), rand::random::<f64>()).norm() * 5.0,
        }
    }

    pub fn draw_body(&self, ctx: &CanvasRenderingContext2d) -> Result<(), JsValue> {
        let red = &JsValue::from_str("#ff0000");
        ctx.set_stroke_style(red);
        ctx.set_fill_style(red);
        ctx.begin_path();
        ctx.set_line_width(2.0);
        ctx.ellipse(self.pos.x, self.pos.y, 2.0, 2.0, 0.0, 0.0, 2.0 * std::f64::consts::PI)?;
        ctx.fill();
        ctx.close_path();
        ctx.stroke();

        Ok(())
    }

    pub fn draw_velocity(&self, ctx: &CanvasRenderingContext2d) -> Result<(), JsValue> {
        ctx.set_stroke_style(&JsValue::from_str("#000000"));
        ctx.set_line_width(1.0);
        ctx.begin_path();
        ctx.move_to(self.pos.x, self.pos.y);
        let n = self.vel.norm() * 10.0;
        ctx.line_to(self.pos.x + n.x, self.pos.y + n.y);
        ctx.close_path();
        ctx.stroke();

        Ok(())
    }

    pub fn draw_neighbourhood(&self, ctx: &CanvasRenderingContext2d) -> Result<(), JsValue> {
        let area = NEIGHBOURHOOD_AREA.lock().unwrap();

        ctx.set_stroke_style(&JsValue::from("#b2beb5"));
        ctx.begin_path();
        ctx.set_line_width(1.0);
        ctx.ellipse(self.pos.x, self.pos.y, *area, *area, 0.0, 0.0, 2.0 * std::f64::consts::PI)?;
        ctx.close_path();
        ctx.stroke();

        Ok(())
    }

    pub fn step(&mut self, bs: &[Boid], dt: f64, neighbourhood_area: f64, area_width: f64, area_height: f64) {
        let mut separation_steer = Vec2::zero(); // Collision avoidance
        let mut cohesion_steer = Vec2::zero();   // Velocity matching
        let mut alignment_steer = Vec2::zero();  // Flock centring

        let mut state = NEIGHBOURHOOD_AREA.lock().unwrap();
        *state = neighbourhood_area;

        let mut count: usize = 0;
        for b in bs {
            let diff = self.pos - b.pos;
            let dist = diff.mag();

            if dist > neighbourhood_area {
                continue;
            }

            count += 1;
            separation_steer += diff.norm() * f64::exp(-dist * 1e-4);
            cohesion_steer += b.vel;
            alignment_steer += b.pos - self.pos;
        }

        if count > 0 {
            alignment_steer /= count as f64;
            cohesion_steer /= count as f64;
        }

        // This is the target velocity we want to steer to
        let goal = separation_steer + cohesion_steer + alignment_steer;

        // Update the velocity
        self.vel += goal * dt * 10.0;
        if self.vel.mag() > 10.0 {
            self.vel = self.vel.norm() * 10.0;
        }

        // Change the position
        self.pos += self.vel * dt * 10.0;

        // Ensure position is in bounds
        self.pos.x = (self.pos.x + area_width) % area_height;
        self.pos.y = (self.pos.y + area_width) % area_height;
    }
}
