_rotation = random(360);
vx = Math.random() * 10 - 5;
vy = Math.random() * 5 + 1;
_X = _X + (Math.random() * 2 - 1);
_Y = _Y + Math.random() * 2;
fx._alpha = 0;
_root.playsound("jetpack.wav");
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      if(fx._alpha < 100)
      {
         fx._alpha += 15;
      }
      _X = _X + vx;
      _Y = _Y + vy;
      vx *= 0.9;
      _xscale = _xscale - 7;
      _yscale = _xscale;
      if(_Y >= 900 || _root.deleteeverything || _xscale <= 10)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
