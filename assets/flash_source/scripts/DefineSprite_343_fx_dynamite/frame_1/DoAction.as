vx = Math.random() * 10 - 5;
vy = Math.random() * 4 - 2;
_X = _X + vx * 1;
_Y = _Y + vy * 1;
rotationspeed = Math.random() * 10 + 5;
if(random(2) == 0)
{
   rotationspeed *= -1;
}
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      _X = _X + vx;
      vx += vx * -0.8;
      _Y = _Y + vy;
      _xscale = _xscale - 5;
      _yscale = _xscale;
      _rotation = _rotation + rotationspeed;
      _alpha = _alpha - 20;
      if(_alpha <= 1 || _root.deleteeverything)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
