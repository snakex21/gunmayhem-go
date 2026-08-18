_X = _X + (random(20) - 10);
_Y = _Y - random(20);
vx = Math.random() * 20 - 10;
vy = Math.random() * 10 - 20;
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
      _rotation = _rotation + rotationspeed;
      _xscale = _xscale - 6;
      _yscale = _xscale;
      vy += 1.2;
      if(_Y >= 900 || _alpha <= 1 || _root.deleteeverything || _xscale <= 1)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
