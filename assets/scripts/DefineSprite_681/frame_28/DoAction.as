facing = _parent.facing;
xx = _parent._x;
yy = _parent._y;
_root.CP("wep_ducky",xx + 35 * facing,yy - 25,0,Math.abs(_parent.vx + facing) * facing);
_parent.offhandammo -= 1;
