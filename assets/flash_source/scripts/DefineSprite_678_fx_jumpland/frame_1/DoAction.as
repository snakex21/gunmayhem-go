gotoAndStop(_root.ground._currentframe);
vx = Math.random() * 24 - 12;
vy = Math.random() * 1 - 1;
_alpha = Math.round(Math.abs(vx) + 10) * 4;
_xscale = _xscale + random(80);
_yscale = _xscale;
_X = _X + (random(10) - 5);
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
      _Y = _Y + vy;
      vx *= 0.8;
      _rotation = _rotation + rotationspeed;
      _alpha = _alpha - 3;
      if(_alpha <= 1 || _root.deleteeverything)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
