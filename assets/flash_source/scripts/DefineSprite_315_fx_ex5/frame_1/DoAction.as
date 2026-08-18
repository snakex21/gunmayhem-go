_X = _X + (random(80) - 40);
_Y = _Y + (random(80) - 20);
vx = Math.random() * 10 - 5;
vy = Math.random() * 4 - 5;
_X = _X + vx * 1;
_Y = _Y + vy * 1;
_xscale = _xscale + random(100);
_yscale = _xscale;
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      _X = _X + vx;
      vy -= 0.1;
      vx *= 0.9;
      _Y = _Y + vy;
      _xscale = _xscale - 6;
      _yscale = _xscale;
      if(_xscale <= 10 || _root.deleteeverything)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
